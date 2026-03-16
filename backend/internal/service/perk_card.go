package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"gorm.io/gorm"
)

const errPerkCardNotFound = "权益卡不存在"

type PerkCardService struct {
	perkRepo *repository.PerkCardRepo
	userRepo *repository.UserRepo
}

func NewPerkCardService(perkRepo *repository.PerkCardRepo, userRepo *repository.UserRepo) *PerkCardService {
	return &PerkCardService{
		perkRepo: perkRepo,
		userRepo: userRepo,
	}
}

func (s *PerkCardService) CreatePerkCard(callerID string, req dto.CreatePerkCardRequest) (*dto.PerkCardItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if user.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以发放权益卡")
	}
	if user.PartnerID == nil {
		return nil, errors.New(errPartnerRequired)
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("过期时间不能早于当前时间")
	}

	card := &model.PerkCard{
		FromUserID:  callerID,
		ToUserID:    *user.PartnerID,
		Title:       req.Title,
		Description: req.Description,
		IconType:    req.IconType,
		Status:      model.PerkActive,
		ExpiresAt:   req.ExpiresAt,
	}
	if err := s.perkRepo.Create(card); err != nil {
		return nil, err
	}
	item := toPerkCardItem(*card)
	return &item, nil
}

func (s *PerkCardService) GetPerkCards(callerID string) ([]dto.PerkCardItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}

	var cards []model.PerkCard
	switch user.Role {
	case model.RoleDad:
		cards, err = s.perkRepo.FindByRecipient(callerID, 100)
	case model.RoleMom:
		if user.PartnerID == nil {
			return []dto.PerkCardItem{}, nil
		}
		cards, err = s.perkRepo.FindByIssuer(callerID, *user.PartnerID, 100)
	default:
		return nil, fmt.Errorf("角色无权查看权益卡")
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]dto.PerkCardItem, 0, len(cards))
	for i := range cards {
		// Auto-expire active cards when reading
		if cards[i].Status == model.PerkActive && cards[i].ExpiresAt != nil && cards[i].ExpiresAt.Before(now) {
			cards[i].Status = model.PerkExpired
			if err := s.perkRepo.MarkExpired(cards[i].ID); err != nil {
				log.Printf("[PerkCard] mark expired failed: %v", err)
			}
		}
		items = append(items, toPerkCardItem(cards[i]))
	}

	return items, nil
}

func (s *PerkCardService) UsePerkCard(callerID, cardID string) (*dto.PerkCardItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if user.Role != model.RoleDad {
		return nil, fmt.Errorf("只有爸爸角色可以核销权益卡")
	}

	card, err := s.perkRepo.FindByID(cardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errPerkCardNotFound)
		}
		return nil, err
	}
	if card.ToUserID != callerID {
		return nil, fmt.Errorf("无权操作此权益卡")
	}
	if card.Status != model.PerkActive {
		return nil, fmt.Errorf("权益卡已使用或已失效")
	}

	now := time.Now()
	if card.ExpiresAt != nil && card.ExpiresAt.Before(now) {
		_ = s.perkRepo.MarkExpired(card.ID)
		card.Status = model.PerkExpired
		return nil, fmt.Errorf("权益卡已过期")
	}

	if err := s.perkRepo.MarkUsed(card.ID, now); err != nil {
		return nil, err
	}
	card.Status = model.PerkUsed
	card.UsedAt = &now

	item := toPerkCardItem(*card)
	return &item, nil
}

func toPerkCardItem(card model.PerkCard) dto.PerkCardItem {
	return dto.PerkCardItem{
		ID:          card.ID,
		FromUserID:  card.FromUserID,
		ToUserID:    card.ToUserID,
		Title:       card.Title,
		Description: card.Description,
		IconType:    card.IconType,
		Status:      string(card.Status),
		UsedAt:      card.UsedAt,
		ExpiresAt:   card.ExpiresAt,
		CreatedAt:   card.CreatedAt,
	}
}
