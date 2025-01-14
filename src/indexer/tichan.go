package indexer

import (
	"path"
	"path/filepath"
	"strings"
	"tyson-tap/src/conf"

	"github.com/c2h5oh/datasize"
	"github.com/triole/logseal"
)

func (ind *Indexer) assembleTapIndex() {
	chin := make(chan string, ind.Conf.Threads)
	chout := make(chan TapEntry, ind.Conf.Threads)
	ln := len(ind.DataSources.Paths)
	if ln > 0 {
		for _, pth := range ind.DataSources.Paths {
			switch ind.DataSources.Type {
			case "url":
				ind.Lg.Debug("fetch url", logseal.F{"path": pth})
				go ind.fetchURL(
					pth, ind.DataSources.Params.Endpoint,
					chin, chout,
				)
			default:
				ind.Lg.Debug("read file", logseal.F{"path": pth})
				go ind.readFile(
					pth, ind.DataSources.Params.Endpoint,
					chin, chout,
				)
			}
		}

		c := 0
		for te := range chout {
			te.SortIndex = ind.TapIndex.stringifySortIndex(
				[]interface{}{ind.Util.GetPathDepth(te.Path), te.Path},
			)
			ind.TapIndex = append(ind.TapIndex, te)
			c++
			if c >= ln {
				close(chin)
				close(chout)
				break
			}
		}
	} else {
		ind.Lg.Warn("no data source paths", logseal.F{"data_source": ind.DataSources})
	}
}

func (ind Indexer) fetchURL(pth string, ep conf.Endpoint, chin chan string, chout chan TapEntry) {
	te := TapEntry{Path: pth}
	chin <- te.Path
	if ep.Return.Content || ep.Process.Strategy != "" {
		resp, err := ind.req(ep.Source, ep.RequestMethod)
		te.Content = ind.byteToBody(resp)
		te.Content.Error = err
		if te.Content.Error == nil {
			te.Content = ind.unmarshal(resp, ep)
		}
	}
	chout <- te
	<-chin
}

func (ind Indexer) readFile(pth string, ep conf.Endpoint, chin chan string, chout chan TapEntry) {
	chin <- pth
	te := TapEntry{FullPath: pth, Path: pth}
	basepth := path.Base(te.FullPath)
	if !strings.EqualFold(basepth, ep.Source) {
		te.Path = strings.TrimPrefix(
			strings.TrimPrefix(te.Path, ep.Source), string(filepath.Separator),
		)
	}
	fileSize := ind.Util.GetFileSize(te.FullPath)
	if ep.Return.Size {
		te.Size = fileSize
	}
	if ep.MaxReturnSizeBytes > fileSize {
		if ep.Return.Content || ep.Return.SplitMarkdownFrontMatter {
			te.Content = ind.readFileContent(te.FullPath, ep)
		}
	} else {
		ind.Lg.Trace(
			"do not return file content, size limit exceeded",
			logseal.F{
				"path":      te.FullPath,
				"file_size": datasize.ByteSize(fileSize).HumanReadable(),
				"max_size":  datasize.ByteSize(ep.MaxReturnSizeBytes).HumanReadable(),
			},
		)
	}
	if ep.Return.SplitPath {
		te.SplitPath = strings.Split(te.Path, string(filepath.Separator))
	}
	if ep.Return.Created {
		te.Created = ind.Util.GetFileCreated(te.FullPath)
	}
	if ep.Return.LastMod {
		te.LastMod = ind.Util.GetFileLastMod(te.FullPath)
	}
	chout <- te
	<-chin
}
