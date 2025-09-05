package entities

type PullRequest struct {
	Title string
	Body  string
}

func (p *PullRequest) IsValid() bool {
	return p.Title != "" && p.Body != ""
}

func NewPullRequest(title, body string) *PullRequest {
	return &PullRequest{
		Title: title,
		Body:  body,
	}
}

func (p *PullRequest) GetTitle() string {
	return p.Title
}

func (p *PullRequest) GetBody() string {
	return p.Body
}
