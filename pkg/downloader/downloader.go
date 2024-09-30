package downloader

type Info struct {
	VCS  string `json:"VCS"`
	URL  string `json:"URL"`
	Hash string `json:"Hash"`
	Ref  string `json:"Ref"`
}

type DownloadFn func(cacheDir string) (string, error)

type Downloader interface {
	Discover(string, string) (DownloadFn, Info, error)
}
