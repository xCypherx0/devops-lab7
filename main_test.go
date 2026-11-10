package main

import (
	"context"
	"testing"
)

func TestCarLifecycle(t *testing.T) {
	repo := &repository{cars: make(map[string]Car)}
	ctx := context.Background()
	id, err := repo.Create(ctx, Car{Brand: "Toyota", Model: "Camry", Year: 2020, Price: 2500000})
	if err != nil {
		t.Fatal(err)
	}
	car, err := repo.GetByID(ctx, id)
	if err != nil || car.Brand != "Toyota" || car.Sold {
		t.Fatalf("unexpected car: %+v, %v", car, err)
	}
	if err := repo.MarkSold(ctx, id); err != nil {
		t.Fatal(err)
	}
	car, err = repo.GetByID(ctx, id)
	if err != nil || !car.Sold {
		t.Fatalf("expected sold car: %+v, %v", car, err)
	}
}

func BenchmarkGetByID(b *testing.B) {
	repo := &repository{cars: make(map[string]Car)}
	ctx := context.Background()
	id, _ := repo.Create(ctx, Car{Brand: "Toyota", Model: "Camry"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.GetByID(ctx, id); err != nil {
			b.Fatal(err)
		}
	}
}
