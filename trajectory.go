package rocket

import (
	"fmt"
	"strings"
)

// TrajectoryKind identifies the kind of actor tracked by an ActorTrajectory.
type TrajectoryKind string

const (
	TrajectoryKindCar  TrajectoryKind = "car"
	TrajectoryKindBall TrajectoryKind = "ball"
)

// PositionSample records a known actor position at a replay frame.
type PositionSample struct {
	Frame    int
	Time     float32
	Position Vector3f
}

// ActorTrajectory contains the known positions for one actor lifetime.
// Instance distinguishes actor ID reuse; it starts at zero for each actor ID.
type ActorTrajectory struct {
	ActorID    int32
	Instance   int
	Kind       TrajectoryKind
	ObjectName string
	Samples    []PositionSample
}

// Trajectories derives car and ball positions from decoded network frames.
// It records only positions present in spawn data or rigid-body updates and
// starts a new trajectory when a network actor ID is reused.
func (replay *Replay) Trajectories() ([]ActorTrajectory, error) {
	if replay == nil {
		return nil, fmt.Errorf("replay is nil")
	}

	trajectories := make([]ActorTrajectory, 0)
	active := make(map[int32]int)
	instances := make(map[int32]int)
	for frameIndex, frame := range replay.Frames {
		for _, actor := range frame.Actors {
			if actor.Spawn != nil {
				delete(active, actor.ActorID)
				objectIndex := actor.Spawn.ObjectIndex
				if objectIndex < 0 || uint64(objectIndex) >= uint64(len(replay.Objects)) {
					return nil, fmt.Errorf("frame %d actor %d has invalid spawn object index %d",
						frameIndex, actor.ActorID, objectIndex)
				}
				objectName := replay.Objects[objectIndex]
				kind := trajectoryKind(objectName)
				if kind != "" {
					index := len(trajectories)
					trajectories = append(trajectories, ActorTrajectory{
						ActorID: actor.ActorID, Instance: instances[actor.ActorID],
						Kind: kind, ObjectName: objectName,
					})
					instances[actor.ActorID]++
					active[actor.ActorID] = index
					if actor.Spawn.Location != nil {
						location := actor.Spawn.Location
						trajectories[index].Samples = append(trajectories[index].Samples, PositionSample{
							Frame: frameIndex, Time: frame.Time,
							Position: Vector3f{
								X: float32(location.X) / 100,
								Y: float32(location.Y) / 100,
								Z: float32(location.Z) / 100,
							},
						})
					}
				}
			}

			index, tracked := active[actor.ActorID]
			if tracked && !actor.Deleted {
				for _, update := range actor.Attributes {
					body, ok := update.Value.(RigidBodyState)
					if !ok {
						continue
					}
					trajectories[index].Samples = append(trajectories[index].Samples, PositionSample{
						Frame: frameIndex, Time: frame.Time, Position: body.Location,
					})
				}
			}
			if actor.Deleted {
				delete(active, actor.ActorID)
			}
		}
	}
	return trajectories, nil
}

func trajectoryKind(objectName string) TrajectoryKind {
	switch {
	case strings.HasPrefix(objectName, "Archetypes.Car."):
		return TrajectoryKindCar
	case strings.HasPrefix(objectName, "Archetypes.Ball."):
		return TrajectoryKindBall
	default:
		return ""
	}
}
