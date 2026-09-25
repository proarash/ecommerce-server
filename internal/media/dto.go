package media

type UploadResponse struct {
	ID        uint   `json:"id"`
	URL       string `json:"url"`
	MediaType string `json:"media_type"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	FileName  string `json:"file_name"`
}
