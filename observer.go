package game

type EventType string

type Event struct {
	Name EventType   // What happened
	Data interface{} // Optional context
}

const (
	EventBulletFired EventType = "BulletFired"
	EventExplosion   EventType = "Explosion"
	EventPlayerHit   EventType = "PlayerHit"
	// this is not really implemented
	// should be used for 'forceful return to lobby'
	// probably...
	EventBackToLobby EventType = "BackToLobby"
	EventGameOver    EventType = "GameOver"
	EventNewMatch    EventType = "NewMatch"
	EventNewRound    EventType = "NewRound"
)

type Observer interface {
	OnEvent(event Event)
}

type Subject interface {
	Register(observer Observer)
	Deregister(observer Observer)
	Notify(event Event)
}

type GenericSubject struct {
	Subject
	observers []Observer
}

func (gs *GenericSubject) Register(observer Observer) {
	gs.observers = append(gs.observers, observer)
}
func (gs *GenericSubject) Deregister(observer Observer) {
	// not yet implemented
}
func (gs *GenericSubject) Notify(event Event) {
	for _, observer := range gs.observers {
		observer.OnEvent(event)
	}
}
