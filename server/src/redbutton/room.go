package redbutton

import (
	"redbutton/api"
	"sync"
)

type RoomListenerMessageHandler func(msg interface{}) error

type VoterId string

// an entity that is interested in room events
// receives notifications via provided message handler
type RoomListener struct {
	voterId        VoterId
	messageHandler RoomListenerMessageHandler
}

// notifies this room listener that there's a new event
func (this *RoomListener) newEvent(message interface{}) {
	this.messageHandler(message)
}

type Room struct {
	sync.RWMutex
	id        string
	name      string
	owner     string
	listeners map[*RoomListener]bool
	voters    map[VoterId]bool
}

func NewRoom() *Room {
	return &Room{
		listeners: map[*RoomListener]bool{},
		voters:    map[VoterId]bool{},
	}
}

// TODO: http layer ideally should not be in this file at all
func roomAsJson(room *Room) *api.RoomInfo {
	info := api.RoomInfo{}
	info.Id = room.id
	info.RoomName = room.name
	info.RoomOwner = room.owner
	return &info
}

func (this *Room) ensureVoterPresent(voterId VoterId) {
	// voters are by default happy
	if _, ok := this.voters[voterId]; !ok {
		this.voters[voterId] = true
	}
}

func (this *Room) createListener(voterId VoterId, handler RoomListenerMessageHandler) *RoomListener {

	listener := &RoomListener{voterId: voterId, messageHandler: handler}

	this.Lock()
	this.listeners[listener] = true
	this.ensureVoterPresent(voterId)
	this.Unlock()

	this.notifyStatusChanged()
	return listener
}

func (this *Room) unregisterListener(listener *RoomListener) {
	this.Lock()
	delete(this.listeners, listener)
	this.Unlock()

	this.notifyStatusChanged()
}

// builds and sends out a RoomStatusChangeEvent to this room
func (this *Room) notifyStatusChanged() {
	this.RLock()

	// collect IDs of active voters
	activeVoters := map[VoterId]bool{}
	for listener, _ := range this.listeners {
		activeVoters[listener.voterId] = true
	}

	// count votes of active unhappy voters
	numUnhappy := 0
	for voterId, happy := range this.voters {
		if _, ok := activeVoters[voterId]; ok {
			if !happy {
				numUnhappy++
			}
		}
	}

	msg := api.RoomStatusChangeEvent{
		RoomName:        this.name,
		NumFlags:        numUnhappy,
		NumParticipants: len(activeVoters),
	}
	this.RUnlock()
	this.broadcastMessage(msg)
}

// broadcasts a message to all room listeners
func (this *Room) broadcastMessage(message interface{}) {
	this.RLock()
	defer this.RUnlock()
	for listener, _ := range this.listeners {
		go listener.newEvent(message)
	}
}

func (this *Room) setHappy(voterId VoterId, happy bool) {
	this.Lock()
	this.voters[voterId] = happy
	this.Unlock()

	this.notifyStatusChanged()
}

func (this *Room) getVoterStatus(voterId VoterId) *api.VoterStatus {
	result := api.VoterStatus{}
	this.ensureVoterPresent(voterId)
	result.Happy = this.voters[voterId]
	return &result
}