package inventory

import (
	"atlas-character/kafka"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitEquipItemCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, destination int16) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvCommandTopicEquipItem))
	return func(characterId uint32, source int16, destination int16) {
		event := &equipItemCommand{
			Tenant:      tenant,
			CharacterId: characterId,
			Source:      source,
			Destination: destination,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitUnequipItemCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvCommandTopicUnequipItem))
	return func(characterId uint32, source int16) {
		event := &unequipItemCommand{
			Tenant:      tenant,
			CharacterId: characterId,
			Source:      source,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitItemGainEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, itemId uint32, quantity uint32, slot int16) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicItemGain))
	return func(characterId uint32, itemId uint32, quantity uint32, slot int16) {
		event := &gainItemEvent{
			Tenant:      tenant,
			CharacterId: characterId,
			ItemId:      itemId,
			Quantity:    quantity,
			Slot:        slot,
		}
		p(producer.CreateKey(int(characterId)), event)
	}
}

func emitItemEquipped(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, itemId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicEquipChanged))
	return func(characterId uint32, itemId uint32) {
		e := &equipChangedEvent{
			Tenant:      tenant,
			CharacterId: characterId,
			Change:      "EQUIPPED",
			ItemId:      itemId,
		}
		p(producer.CreateKey(int(characterId)), e)
	}
}

func emitItemUnequipped(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, itemId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicEquipChanged))
	return func(characterId uint32, itemId uint32) {
		e := &equipChangedEvent{
			Tenant:      tenant,
			CharacterId: characterId,
			Change:      "UNEQUIPPED",
			ItemId:      itemId,
		}
		p(producer.CreateKey(int(characterId)), e)
	}
}
