package poll

import "context"

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.cache.Delete(ctx, id)
	return nil
}
