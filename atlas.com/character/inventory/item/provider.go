package item

import (
	"atlas-character/database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getById(tenantId uuid.UUID, id uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, ID: id})
	}
}

func getForCharacter(tenantId uuid.UUID, inventoryId uint32, itemId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{TenantId: tenantId, InventoryId: inventoryId, ItemId: itemId})
	}
}

func getBySlot(tenantId uuid.UUID, inventoryId uint32, slot int16) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{TenantId: tenantId, InventoryId: inventoryId, Slot: slot})
	}
}

func entityByInventory(tenantId uuid.UUID) func(inventoryId uint32) database.EntityProvider[[]entity] {
	return func(inventoryId uint32) database.EntityProvider[[]entity] {
		return func(db *gorm.DB) model.Provider[[]entity] {
			return database.SliceQuery[entity](db, &entity{TenantId: tenantId, InventoryId: inventoryId})
		}
	}
}
