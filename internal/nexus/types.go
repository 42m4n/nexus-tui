package nexus

type Repository struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
}

type Asset struct {
	ID           string `json:"id"`
	Repository   string `json:"repository"`
	Format       string `json:"format"`
	Path         string `json:"path"`
	DownloadURL  string `json:"downloadUrl"`
	ContentType  string `json:"contentType"`
	LastModified string `json:"lastModified"`
	FileSize     int64  `json:"fileSize"`
	Checksum     struct {
		SHA1 string `json:"sha1"`
		MD5  string `json:"md5"`
	} `json:"checksum"`
}

type Component struct {
	ID         string  `json:"id"`
	Repository string  `json:"repository"`
	Format     string  `json:"format"`
	Group      string  `json:"group"`
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	Assets     []Asset `json:"assets"`
}

type Task struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Message       string `json:"message"`
	CurrentState  string `json:"currentState"`
	LastRunResult string `json:"lastRunResult"`
	NextRun       string `json:"nextRun"`
	LastRun       string `json:"lastRun"`
}

type User struct {
	UserID        string   `json:"userId"`
	FirstName     string   `json:"firstName"`
	LastName      string   `json:"lastName"`
	Email         string   `json:"emailAddress"`
	Source        string   `json:"source"`
	Status        string   `json:"status"`
	ReadOnly      bool     `json:"readOnly"`
	ExternalRoles []string `json:"externalRoles"`
}

type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Privileges  []string `json:"privileges"`
	Roles       []string `json:"roles"`
}

type Privilege struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type BlobStore struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	BlobCount      int64  `json:"blobCount"`
	TotalSize      int64  `json:"totalSize"`
	AvailableSpace int64  `json:"availableSpace"`
}

type page[T any] struct {
	Items             []T    `json:"items"`
	ContinuationToken string `json:"continuationToken"`
}
