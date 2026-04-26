package gh

type ProjectSummary struct {
	ID     string
	Number int
	Title  string
	URL    string
}

type SingleSelectOption struct {
	ID   string
	Name string
}

type SingleSelectField struct {
	ID      string
	Name    string
	Options []SingleSelectOption
}

type Project struct {
	ID     string
	Title  string
	Number int
	URL    string
	Status SingleSelectField
	Items  []Item
}

type ItemContentType string

const (
	ContentIssue       ItemContentType = "Issue"
	ContentPullRequest ItemContentType = "PullRequest"
	ContentDraftIssue  ItemContentType = "DraftIssue"
)

type Item struct {
	ID             string
	ContentType    ItemContentType
	Title          string
	Body           string
	URL            string
	Number         int
	Assignees      []string
	Labels         []string
	StatusOptionID string
}
