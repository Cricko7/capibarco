package postgres

import (
	"strings"
	"testing"
)

func TestIdempotencyLookupQueryQualifiesSelectedAnimalColumns(t *testing.T) {
	query := selectAnimalSQL + ` JOIN idempotency_keys idem ON idem.animal_id = animals.animal_id WHERE idem.key = $1`

	if strings.Contains(query, "\n\tanimal_id,") {
		t.Fatalf("idempotency lookup query selects ambiguous animal_id column:\n%s", query)
	}
	if !strings.Contains(query, "\n\tanimals.animal_id,") {
		t.Fatalf("idempotency lookup query should qualify selected animal columns:\n%s", query)
	}
}
