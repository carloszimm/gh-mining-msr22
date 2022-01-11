package types

type Info struct {
	Owner              string `json:"owner"`
	RepositoryName     string `json:"repoName"`
	RepositoryFullName string `json:"repoFullName"`
	Branch             string `json:"branch"`
	FileName           string `json:"fileName"`
	FileSize           int    `json:"fileSize"`
	ArchiveUrl         string `json:"url"`
}

type InfoFile struct {
	Owner              string `json:"-"`
	RepositoryName     string `json:"-"`
	RepositoryFullName string `json:"-"`
	Branch             string `json:"-"`
	FileName           string `json:"fileName"`
	FileSize           int    `json:"-"`
	ArchiveUrl         string `json:"-"`
}
