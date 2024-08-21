package item

import (
	"atlas-character/asset"
	"atlas-character/database"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

func ByInventoryProvider(db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) model.Provider[[]Model] {
	return func(inventoryId uint32) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getByInventory(tenant.Id, inventoryId), makeModel)
	}
}

func GetByInventory(db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return ByInventoryProvider(db, tenant)(inventoryId)()
	}
}

func BySlotProvider(db *gorm.DB) func(tenant tenant.Model) func(inventoryId uint32) func(slot int16) model.Provider[Model] {
	return func(tenant tenant.Model) func(inventoryId uint32) func(slot int16) model.Provider[Model] {
		return func(inventoryId uint32) func(slot int16) model.Provider[Model] {
			return func(slot int16) model.Provider[Model] {
				return database.ModelProvider[Model, entity](db)(getBySlot(tenant.Id, inventoryId, slot), makeModel)
			}
		}
	}
}

func GetBySlot(db *gorm.DB, tenant tenant.Model) func(inventoryId uint32, slot int16) (Model, error) {
	return func(inventoryId uint32, slot int16) (Model, error) {
		return BySlotProvider(db)(tenant)(inventoryId)(slot)()
	}
}

func GetById(db *gorm.DB, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return database.ModelProvider[Model, entity](db)(getById(tenant.Id, id), makeModel)()
	}
}

func ByItemIdProvider(db *gorm.DB) func(tenant tenant.Model) func(inventoryId uint32) func(itemId uint32) model.Provider[[]Model] {
	return func(tenant tenant.Model) func(inventoryId uint32) func(itemId uint32) model.Provider[[]Model] {
		return func(inventoryId uint32) func(itemId uint32) model.Provider[[]Model] {
			return func(itemId uint32) model.Provider[[]Model] {
				return database.ModelSliceProvider[Model, entity](db)(getForCharacter(tenant.Id, inventoryId, itemId), makeModel)
			}
		}
	}
}

func GetByItemId(db *gorm.DB, tenant tenant.Model) func(inventoryId uint32, itemId uint32) ([]Model, error) {
	return func(inventoryId uint32, itemId uint32) ([]Model, error) {
		return ByItemIdProvider(db)(tenant)(inventoryId)(itemId)()
	}
}

func UpdateSlot(db *gorm.DB) func(tenant tenant.Model) func(id uint32, slot int16) error {
	return func(tenant tenant.Model) func(id uint32, slot int16) error {
		return func(id uint32, slot int16) error {
			return updateSlot(db, id, slot)
		}
	}
}

func UpdateQuantity(db *gorm.DB, tenant tenant.Model) func(id uint32, quantity uint32) error {
	return func(id uint32, quantity uint32) error {
		i, err := GetById(db, tenant)(id)
		if err != nil {
			return err
		}
		return updateQuantity(db, i.Id(), quantity)
	}
}

func CreateItem(db *gorm.DB, tenant tenant.Model) asset.CharacterAssetCreator {
	return func(characterId uint32) asset.InventoryAssetCreator {
		return func(inventoryId uint32, inventoryType int8) asset.ItemCreator {
			return func(itemId uint32) asset.Creator {
				return func(quantity uint32) model.Provider[asset.Asset] {
					slot, err := slottable.GetNextFreeSlot(model.SliceMap(ByInventoryProvider(db, tenant)(inventoryId), ToSlottable))
					if err != nil {
						return model.ErrorProvider[asset.Asset](err)
					}
					i, err := createItem(db, tenant, inventoryId, itemId, quantity, slot)
					if err != nil {
						return model.ErrorProvider[asset.Asset](err)
					}
					return model.FixedProvider[asset.Asset](i)
				}
			}
		}
	}
}

func ToAsset(m Model) (asset.Asset, error) {
	return m, nil
}

func ToSlottable(m Model) (asset.Slottable, error) {
	return m, nil
}

func RemoveItem(db *gorm.DB) func(characterId uint32, id uint32) error {
	return func(characterId uint32, id uint32) error {
		return remove(db, characterId, id)
	}
}

func DeleteById(db *gorm.DB) func(tenant tenant.Model) model.Operator[uint32] {
	return func(tenant tenant.Model) model.Operator[uint32] {
		return func(id uint32) error {
			return deleteById(db, tenant.Id, id)
		}
	}
}
