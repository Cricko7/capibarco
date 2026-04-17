package postgres

import (
	"reflect"
	"strings"
	"testing"

	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	"github.com/petmatch/petmatch/internal/feed"
	"google.golang.org/protobuf/proto"
)

func TestCandidateRowRoundTripPreservesProjection(t *testing.T) {
	candidate := feed.Candidate{
		Animal: &animalv1.AnimalProfile{
			AnimalId:       "animal-1",
			OwnerProfileId: "owner-1",
			Species:        animalv1.Species_SPECIES_DOG,
			Status:         animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE,
			Location:       &commonv1.Address{City: "Moscow"},
			Visibility:     commonv1.Visibility_VISIBILITY_PUBLIC,
			Boosted:        true,
			Vaccinated:     true,
			Sterilized:     true,
		},
		OwnerDisplayName:   "Ada",
		OwnerAverageRating: 4.8,
		RankingReasons:     []string{"active boost", "analytics engagement"},
		ScoreComponents:    map[string]float64{"boost": 1, "ctr": 0.4},
		DistanceKM:         9,
		OwnerHidden:        true,
		OwnerBlocked:       false,
	}

	row, err := newCandidateRow(candidate)
	if err != nil {
		t.Fatalf("newCandidateRow returned error: %v", err)
	}
	got, err := row.candidate()
	if err != nil {
		t.Fatalf("candidate returned error: %v", err)
	}

	if !proto.Equal(got.Animal, candidate.Animal) {
		t.Fatalf("animal mismatch: got %#v want %#v", got.Animal, candidate.Animal)
	}
	if !reflect.DeepEqual(got.RankingReasons, candidate.RankingReasons) {
		t.Fatalf("ranking reasons = %v, want %v", got.RankingReasons, candidate.RankingReasons)
	}
	if !reflect.DeepEqual(got.ScoreComponents, candidate.ScoreComponents) {
		t.Fatalf("score components = %v, want %v", got.ScoreComponents, candidate.ScoreComponents)
	}
	if got.OwnerDisplayName != candidate.OwnerDisplayName ||
		got.OwnerAverageRating != candidate.OwnerAverageRating ||
		got.DistanceKM != candidate.DistanceKM ||
		got.OwnerHidden != candidate.OwnerHidden ||
		got.OwnerBlocked != candidate.OwnerBlocked {
		t.Fatalf("projection mismatch: got %#v want %#v", got, candidate)
	}
}

func TestListCandidatesSQLIncludesFilterPredicates(t *testing.T) {
	city := "Moscow"
	boostedOnly := true
	vaccinated := true
	filter := &animalv1.AnimalFilter{
		Species:     []animalv1.Species{animalv1.Species_SPECIES_DOG},
		Statuses:    []animalv1.AnimalStatus{animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE},
		City:        &city,
		BoostedOnly: &boostedOnly,
		Vaccinated:  &vaccinated,
	}

	query, args := listCandidatesSQL(filter)

	for _, predicate := range []string{
		"species = ANY($1)",
		"status = ANY($2)",
		"city = $3",
		"boosted = $4",
		"vaccinated = $5",
	} {
		if !strings.Contains(query, predicate) {
			t.Fatalf("query %q does not contain predicate %q", query, predicate)
		}
	}
	if len(args) != 5 {
		t.Fatalf("args len = %d, want 5: %v", len(args), args)
	}
}

func TestMigrationFilesAreOrdered(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations returned error: %v", err)
	}

	if len(migrations) != 1 {
		t.Fatalf("migration count = %d, want 1", len(migrations))
	}
	if migrations[0].Version != 1 {
		t.Fatalf("migration version = %d, want 1", migrations[0].Version)
	}
	if migrations[0].Name != "001_init.sql" {
		t.Fatalf("migration name = %q, want 001_init.sql", migrations[0].Name)
	}
	for _, statement := range []string{
		"CREATE TABLE IF NOT EXISTS feed_candidates",
		"CREATE TABLE IF NOT EXISTS feed_served_cards",
		"CREATE TABLE IF NOT EXISTS feed_schema_migrations",
	} {
		if !strings.Contains(migrations[0].SQL, statement) {
			t.Fatalf("migration SQL does not contain %q", statement)
		}
	}
}
