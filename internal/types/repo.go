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
