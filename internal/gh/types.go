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

// ProjectSpec selects a project either by exact title or by project number.
// At least one of Title / Number must be set; Number takes precedence.
type ProjectSpec struct {
	Title  string
	Number int
}

// BootstrapResult is what a single GraphQL bootstrap call returns: the project
// with its Status field and the first page of items, plus a cursor to the next
// page (empty if there are no more pages) and the total item count reported by
// GitHub for the connection (used to render % progress while paging).
type BootstrapResult struct {
	Project    *Project
	NextCursor string
	TotalItems int
}

// ItemsPage is one slice of items appended to an already-bootstrapped project.
// TotalItems is the connection's totalCount as reported on this page.
type ItemsPage struct {
	Items      []Item
	NextCursor string
	TotalItems int
}
