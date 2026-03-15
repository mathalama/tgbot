package usecase

import (
	"context"

	"tgbot/internal/domain"
)

// ManageChannelsUseCase управляет каналами для мониторинга
type ManageChannelsUseCase struct {
	channelRepo domain.ChannelRepository
}

// NewManageChannelsUseCase создает новый use case
func NewManageChannelsUseCase(channelRepo domain.ChannelRepository) *ManageChannelsUseCase {
	return &ManageChannelsUseCase{
		channelRepo: channelRepo,
	}
}

// AddChannel добавляет новый канал
func (uc *ManageChannelsUseCase) AddChannel(ctx context.Context, username string) error {
	channel := &domain.Channel{
		ID:       username,
		Username: username,
		IsActive: true,
	}
	return uc.channelRepo.Save(ctx, channel)
}

// RemoveChannel удаляет канал
func (uc *ManageChannelsUseCase) RemoveChannel(ctx context.Context, username string) error {
	return uc.channelRepo.Delete(ctx, username)
}

// GetAllChannels получает все каналы
func (uc *ManageChannelsUseCase) GetAllChannels(ctx context.Context) ([]*domain.Channel, error) {
	return uc.channelRepo.GetAll(ctx)
}
