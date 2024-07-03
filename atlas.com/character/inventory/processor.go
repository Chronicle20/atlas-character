package inventory

import (
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/inventory/item"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
)

func byCharacterIdProvider(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return model.Fold[entity, Model](getByCharacter(tenant.Id(), characterId)(db), supplier, foldInventory(l, db, tenant))
	}
}

func supplier() (Model, error) {
	return Model{
		equipable: EquipableModel{},
		useable:   ItemModel{},
		setup:     ItemModel{},
		etc:       ItemModel{},
		cash:      ItemModel{},
	}, nil
}

func foldInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(ref Model, ent entity) (Model, error) {
	return func(ref Model, ent entity) (Model, error) {
		switch Type(ent.InventoryType) {
		case TypeValueEquip:
			equipables, err := equipable.GetInInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.equipable = ref.Equipable().SetItems(equipables).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueUse:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.useable = ref.Useable().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueSetup:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.setup = ref.Setup().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueETC:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.etc = ref.ETC().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueCash:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.cash = ref.Cash().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		}
		return ref, errors.New("unknown inventory type")
	}
}

func GetInventories(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return byCharacterIdProvider(l, db, tenant)(characterId)()
	}
}

func GetInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryType string, filters ...ItemFilter) (Model, error) {
	return func(characterId uint32, inventoryType string, filters ...ItemFilter) (Model, error) {
		//if it, ok := GetTypeFromName(inventoryType); ok {
		//	return GetInventoryByTypeVal(l, db, tenant)(characterId, it, filters...)
		//}
		//return nil, errors.New("invalid inventory type")
		return Model{}, nil
	}
}

func GetInventoryByTypeVal(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryType Type, filters ...ItemFilter) (Model, error) {
	return func(characterId uint32, inventoryType Type, filters ...ItemFilter) (Model, error) {
		inv, err := database.ModelProvider[Model, entity](db)(get(tenant.Id(), characterId, inventoryType), makeInventory)()
		if err != nil {
			return Model{}, err
		}

		//items, err := item.GetByInventory(l, db, tenant)(inv.Id())
		//if err != nil {
		//	return Model{}, err
		//}
		//for _, i := range items {
		//	ok := true
		//	for _, filter := range filters {
		//		if !filter(i) {
		//			ok = false
		//			break
		//		}
		//	}
		//	if ok {
		//		//inv = inv.AddItem(i)
		//	}
		//}

		return inv, nil
	}
}

type ItemFilter func(i item.Model) bool

func FilterSlot(slot int16) ItemFilter {
	return func(i item.Model) bool {
		return i.Slot() == slot
	}
}

func FilterItemId(l logrus.FieldLogger, db *gorm.DB, _ opentracing.Span, tenant tenant.Model) func(itemId uint32) ItemFilter {
	return func(itemId uint32) ItemFilter {
		return func(i item.Model) bool {
			ii, err := item.GetById(l, db, tenant)(i.Id())
			if err != nil {
				return false
			}
			return ii.ItemId() == itemId
		}
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, defaultCapacity uint32) (Model, error) {
	return func(characterId uint32, defaultCapacity uint32) (Model, error) {
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, t := range Types {
				_, err := create(db, tenant.Id(), characterId, int8(t), defaultCapacity)
				if err != nil {
					l.WithError(err).Errorf("Unable to create inventory [%d] for character [%d].", t, characterId)
					return err
				}
			}
			return nil
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to create inventory for character [%d]", characterId)
			return Model{}, err
		}

		return GetInventories(l, db, tenant)(characterId)
	}
}

type adjustment struct {
	mode          byte
	itemId        uint32
	inventoryType Type
	quantity      uint32
	slot          int16
	oldSlot       int16
}

func (i adjustment) Mode() byte {
	return i.mode
}

func (i adjustment) ItemId() uint32 {
	return i.itemId
}

func (i adjustment) InventoryType() Type {
	return i.inventoryType
}

func (i adjustment) Quantity() uint32 {
	return i.quantity
}

func (i adjustment) Slot() int16 {
	return i.slot
}

func (i adjustment) OldSlot() int16 {
	return i.oldSlot
}

func CreateItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
	return func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
		var events = make([]adjustment, 0)

		err := db.Transaction(func(tx *gorm.DB) error {
			m, err := GetInventories(l, tx, tenant)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventories for character [%d].", characterId)
				return err
			}
			inv, err := m.GetHolderByType(inventoryType)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			if inventoryType == TypeValueEquip {
				// Create Equipment
				events, err = createEquipable(l, tx, span, tenant)(characterId, inv.Id(), inventoryType, itemId)
				if err != nil {
					l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", quantity, itemId, characterId)
					return err
				}
			} else {
				// Create Item
				events, err = createItem(l, tx, span, tenant)(characterId, inv.Id(), inventoryType, itemId, quantity)
				if err != nil {
					l.WithError(err).Errorf("Unable to create [%d] items [%d] for character [%d].", quantity, itemId, characterId)
					return err
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
		for _, _ = range events {
			//emitInventoryModificationEvent(l, span)(characterId, true, e.Mode(), e.ItemId(), e.InventoryType(), e.Quantity(), e.Slot(), e.OldSlot())
		}
		return err
	}
}

func createEquipable(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32) ([]adjustment, error) {
	var creator itemCreator = equipable.CreateItem(l, db, span, tenant)
	return func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32) ([]adjustment, error) {
		event, err := createNewItem(l)(creator, characterId, inventoryId, inventoryType, itemId, 1)
		if err != nil {
			return make([]adjustment, 0), nil
		}
		return []adjustment{event}, nil
	}
}

func createItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) ([]adjustment, error) {
	var creator itemCreator = item.CreateItem(l, db, tenant)
	return func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) ([]adjustment, error) {
		runningQuantity := quantity
		slotMax := item.MaxInSlot()
		var events = make([]adjustment, 0)

		existingItems, err := item.GetByItemId(l, db, tenant)(inventoryId, itemId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate items [%d] in inventory [%d] for character [%d].", itemId, inventoryType, characterId)
			return events, err
		}
		if len(existingItems) > 0 {
			index := 0
			for runningQuantity > 0 {
				if index < len(existingItems) {
					i := existingItems[index]
					oldQuantity := i.Quantity()

					if oldQuantity < slotMax {
						newQuantity := uint32(math.Min(float64(oldQuantity+runningQuantity), float64(slotMax)))
						runningQuantity = runningQuantity - (newQuantity - oldQuantity)
						err := item.UpdateQuantity(l, db, tenant)(i.Id(), newQuantity)
						if err != nil {
							l.WithError(err).Errorf("Updating the quantity of item [%d] to value [%d].", i.Id(), newQuantity)
						} else {
							events = append(events, adjustment{mode: 1, itemId: itemId, inventoryType: inventoryType, quantity: newQuantity, slot: i.Slot(), oldSlot: 0})
						}
					}
					index++
				} else {
					break
				}
			}
		}
		for runningQuantity > 0 {
			newQuantity := uint32(math.Min(float64(runningQuantity), float64(slotMax)))
			runningQuantity = runningQuantity - newQuantity
			nes, err := createNewItem(l)(creator, characterId, inventoryId, inventoryType, itemId, newQuantity)
			if err != nil {
				return events, err
			}
			events = append(events, nes)
		}
		return events, nil
	}
}

func createNewItem(l logrus.FieldLogger) func(creator itemCreator, characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) (adjustment, error) {
	return func(creator itemCreator, characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) (adjustment, error) {
		i, err := creator(characterId, inventoryId, int8(inventoryType), itemId, quantity)()
		if err != nil {
			l.WithError(err).Errorf("Unable to create item [%d] for character [%d].", itemId, characterId)
			return adjustment{}, err
		}
		return adjustment{mode: 0, itemId: itemId, inventoryType: inventoryType, quantity: quantity, slot: i.Slot(), oldSlot: 0}, nil
	}
}

type itemCreator func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable]
