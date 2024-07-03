package equipable

import (
	"atlas-character/database"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func ByInventoryProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) model.SliceProvider[Model] {
	return func(inventoryId uint32) model.SliceProvider[Model] {
		return database.ModelSliceProvider[Model, entity](db)(getByInventory(tenant.Id(), inventoryId), makeModel)
	}
}

func GetByInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return ByInventoryProvider(l, db, tenant)(inventoryId)()
	}
}

func GetEquipment(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return model.FilteredProvider[Model](ByInventoryProvider(l, db, tenant)(inventoryId), FilterOutInventory)()
	}
}

func GetInInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return model.FilteredProvider[Model](ByInventoryProvider(l, db, tenant)(inventoryId), FilterOutEquipment)()
	}
}

func FilterOutInventory(e Model) bool {
	return e.Slot() < 0
}

func FilterOutEquipment(e Model) bool {
	return e.Slot() > 0
}

func CreateItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable] {
	return func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable] {
		ms, err := GetByInventory(l, db, tenant)(inventoryId)
		if err != nil {
			return model.ErrorProvider[slottable.Slottable](err)
		}
		slot, err := slottable.GetNextFreeSlot(model.SliceMap(model.FixedSliceProvider(ms), slottableTransformer))
		if err != nil {
			return model.ErrorProvider[slottable.Slottable](err)
		}
		i, err := createItem(db, tenant, inventoryId, itemId, slot)
		if err != nil {
			return model.ErrorProvider[slottable.Slottable](err)
		}
		return model.FixedProvider[slottable.Slottable](i)
	}
}

func slottableTransformer(m Model) (slottable.Slottable, error) {
	return m, nil
}
