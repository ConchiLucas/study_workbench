package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/repo"
)

// RewardService 管小红花兑换。日常打卡任务已废弃，答题进度走 study_plans。
type RewardService struct{ repo *repo.Repo }

func NewRewardService(r *repo.Repo) *RewardService { return &RewardService{repo: r} }

func (s *RewardService) List(childID int64) ([]model.Reward, error) {
	var out []model.Reward
	err := s.repo.DB().Where("child_id = ?", childID).Order("cost").Find(&out).Error
	return out, err
}

func (s *RewardService) Redeem(childID, rewardID int64) error {
	return s.repo.Tx(func(tx *gorm.DB) error {
		var reward model.Reward
		if err := tx.Where("id = ? AND child_id = ?", rewardID, childID).First(&reward).Error; err != nil {
			return err
		}
		if reward.Stock <= 0 {
			return errors.New("奖励已兑完")
		}
		var child model.Child
		if err := tx.First(&child, childID).Error; err != nil {
			return err
		}
		if child.Flowers < reward.Cost {
			return errors.New("小红花不够")
		}
		if err := tx.Model(&model.Reward{}).Where("id = ?", rewardID).
			UpdateColumn("stock", gorm.Expr("stock - 1")).Error; err != nil {
			return err
		}
		return s.repo.AddFlowers(tx, childID, -reward.Cost, "redeem", "reward", &reward.ID)
	})
}
