package provisioner

import (
	"io/ioutil"
	"net/http"

	"github.com/richardwilkes/toolbox/errs"
	"github.com/richardwilkes/toolbox/xio"
)

// ArchiveRetriever is used to retrieve the bytes for a zipped archive of
// files that are to be provisioned.
type ArchiveRetriever func() ([]byte, error)

// FallbackArchiveRetriever returns an ArchiveRetriever that will try each of
// the provided ArchiveRetrievers in turn until one succeeds or none are left.
func FallbackArchiveRetriever(retrievers ...ArchiveRetriever) ArchiveRetriever {
	return func() ([]byte, error) {
		var cumulativeErr error
		for _, r := range retrievers {
			data, err := r()
			if err == nil {
				return data, nil
			}
			cumulativeErr = errs.Append(cumulativeErr, err)
		}
		return nil, cumulativeErr
	}
}

// URLArchiveRetriever returns an ArchiveRetriever that will download the
// archive from a URL.
func URLArchiveRetriever(client *http.Client, url string) ArchiveRetriever {
	return func() ([]byte, error) {
		resp, err := client.Get(url)
		if err != nil {
			return nil, errs.Wrap(err)
		}
		defer xio.CloseIgnoringErrors(resp.Body)
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errs.NewfWithCause(err, "Failed to download %s", url)
		}
		if resp.StatusCode != http.StatusOK {
			err = errs.Newf("Attempted download of %s returned status code %d", url, resp.StatusCode)
		}
		return data, err
	}
}

// FileSystemArchiveRetriever returns an ArchiveRetriever that will return the
// archive from a file system.
func FileSystemArchiveRetriever(fs http.FileSystem, path string) ArchiveRetriever {
	return func() ([]byte, error) {
		f, err := fs.Open(path)
		if err != nil {
			return nil, errs.Wrap(err)
		}
		defer xio.CloseIgnoringErrors(f)
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, errs.NewfWithCause(err, "Failed to load %s", path)
		}
		return data, err
	}
}
