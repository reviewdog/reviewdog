package reviewdog

var _ BulkCommentService = &multiCommentService{}

type multiCommentService struct {
	services []CommentService
}

func (m *multiCommentService) Post(c *Comment) error {
	for _, cs := range m.services {
		if err := cs.Post(c); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiCommentService) Flash() error {
	for _, cs := range m.services {
		if bulk, ok := cs.(BulkCommentService); ok {
			if err := bulk.Flash(); err != nil {
				return err
			}
		}
	}
	return nil
}

// MultiCommentService creates a comment service that duplicates its post to
// all the provided comment services.
func MultiCommentService(services ...CommentService) CommentService {
	s := make([]CommentService, len(services))
	copy(s, services)
	return &multiCommentService{services: s}
}
