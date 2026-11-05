package rocket_test

import (
	"testing"

	rocket "github.com/sbauer/go-rocket"
	"github.com/sbauer/go-rocket/parsers"
	"github.com/sbauer/go-rocket/source"
)

func TestReplayTrajectories(t *testing.T) {
	replay := &rocket.Replay{
		Objects: []string{
			"Archetypes.Car.Car_Default",
			"Archetypes.Ball.Ball_Default",
			"Archetypes.Teams.Team0",
		},
		Frames: []rocket.NetworkFrame{
			{
				Time: 1,
				Actors: []rocket.ActorReplication{
					{ActorID: 1, Spawn: &rocket.SpawnedActor{
						ObjectIndex: 0, Location: &rocket.Vector3i{X: 100, Y: -200, Z: 300},
					}},
					{ActorID: 2, Spawn: &rocket.SpawnedActor{ObjectIndex: 1}},
					{ActorID: 3, Spawn: &rocket.SpawnedActor{ObjectIndex: 2}},
				},
			},
			{
				Time: 2,
				Actors: []rocket.ActorReplication{
					{ActorID: 1, Attributes: []rocket.AttributeUpdate{
						{Value: rocket.RigidBodyState{Location: rocket.Vector3f{X: 4, Y: 5, Z: 6}}},
					}},
					{ActorID: 2, Attributes: []rocket.AttributeUpdate{
						{Value: rocket.RigidBodyState{Location: rocket.Vector3f{X: 7, Y: 8, Z: 9}}},
					}},
					{ActorID: 2, Deleted: true},
				},
			},
			{
				Time: 3,
				Actors: []rocket.ActorReplication{
					{ActorID: 2, Spawn: &rocket.SpawnedActor{
						ObjectIndex: 1, Location: &rocket.Vector3i{X: 1000},
					}},
				},
			},
			{
				Time: 4,
				Actors: []rocket.ActorReplication{
					{ActorID: 2, Attributes: []rocket.AttributeUpdate{
						{Value: rocket.RigidBodyState{Location: rocket.Vector3f{X: 11}}},
					}},
				},
			},
		},
	}

	trajectories, err := replay.Trajectories()
	if err != nil {
		t.Fatal(err)
	}
	if len(trajectories) != 3 {
		t.Fatalf("got %d trajectories, want 3: %+v", len(trajectories), trajectories)
	}
	car := trajectories[0]
	if car.ActorID != 1 || car.Instance != 0 || car.Kind != rocket.TrajectoryKindCar ||
		len(car.Samples) != 2 || car.Samples[0].Position != (rocket.Vector3f{X: 1, Y: -2, Z: 3}) ||
		car.Samples[1].Frame != 1 || car.Samples[1].Time != 2 ||
		car.Samples[1].Position != (rocket.Vector3f{X: 4, Y: 5, Z: 6}) {
		t.Fatalf("unexpected car trajectory: %+v", car)
	}
	firstBall, secondBall := trajectories[1], trajectories[2]
	if firstBall.ActorID != 2 || firstBall.Instance != 0 || firstBall.Kind != rocket.TrajectoryKindBall ||
		len(firstBall.Samples) != 1 || firstBall.Samples[0].Position != (rocket.Vector3f{X: 7, Y: 8, Z: 9}) {
		t.Fatalf("unexpected first ball trajectory: %+v", firstBall)
	}
	if secondBall.ActorID != 2 || secondBall.Instance != 1 || secondBall.Kind != rocket.TrajectoryKindBall ||
		len(secondBall.Samples) != 2 || secondBall.Samples[0].Frame != 2 ||
		secondBall.Samples[0].Position != (rocket.Vector3f{X: 10}) ||
		secondBall.Samples[1].Position != (rocket.Vector3f{X: 11}) {
		t.Fatalf("unexpected reused ball trajectory: %+v", secondBall)
	}
}

func TestReplayTrajectoriesRejectInvalidSpawnObjectIndex(t *testing.T) {
	replay := &rocket.Replay{
		Frames: []rocket.NetworkFrame{{Actors: []rocket.ActorReplication{
			{ActorID: 9, Spawn: &rocket.SpawnedActor{ObjectIndex: 0}},
		}}},
	}
	if _, err := replay.Trajectories(); err == nil {
		t.Fatal("expected invalid object index error")
	}
}

func TestReplayTrajectoriesFromFixture(t *testing.T) {
	replay, err := parsers.ParseWithOptions(source.FromFile("samples/first.replay"),
		parsers.ParseOptions{DecodeNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	trajectories, err := replay.Trajectories()
	if err != nil {
		t.Fatal(err)
	}
	var cars, balls, samples int
	for _, trajectory := range trajectories {
		samples += len(trajectory.Samples)
		switch trajectory.Kind {
		case rocket.TrajectoryKindCar:
			cars++
		case rocket.TrajectoryKindBall:
			balls++
		}
	}
	if cars == 0 || balls == 0 || samples == 0 {
		t.Fatalf("fixture yielded cars=%d balls=%d samples=%d", cars, balls, samples)
	}
}

func TestNilReplayTrajectories(t *testing.T) {
	if _, err := (*rocket.Replay)(nil).Trajectories(); err == nil {
		t.Fatal("expected nil replay error")
	}
}
