package hyperex

import (
	"context"
	"fmt"
	"testing"
)

// =============================================================================
// REAL-WORLD SCENARIO BENCHMARKS
// =============================================================================

func BenchmarkScenario_BusinessRule(b *testing.B) {
	// Typical business rule: check user eligibility
	program := MustCompile(`user.age >= 18 && user.active && user.balance > 0 && !(user.blocked)`)
	env := map[string]any{
		"user": map[string]any{
			"age":     int64(25),
			"active":  true,
			"balance": int64(100),
			"blocked": false,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_Pricing(b *testing.B) {
	// Pricing calculation with discounts
	program := MustCompile(`
		price * quantity * (1 - discount / 100) + (quantity > 10 ? 0 : shipping)
	`)
	env := map[string]any{
		"price":    float64(29.99),
		"quantity": int64(5),
		"discount": float64(10),
		"shipping": float64(5.99),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_Validation(b *testing.B) {
	// Form validation rules
	program := MustCompile(`
		len(email) > 0 && contains(email, "@") && 
		len(password) >= 8 && 
		age >= 13 && age <= 120
	`)
	env := map[string]any{
		"email":    "user@example.com",
		"password": "securepass123",
		"age":      int64(25),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_DataTransform(b *testing.B) {
	// Data transformation
	program := MustCompile(`
		{
			fullName: user.firstName + " " + user.lastName,
			isAdult: user.age >= 18,
			score: (quiz1 + quiz2 + quiz3) / 3
		}
	`)
	env := map[string]any{
		"user": map[string]any{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       int64(25),
		},
		"quiz1": float64(85),
		"quiz2": float64(90),
		"quiz3": float64(88),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_ArrayProcessing(b *testing.B) {
	// Array aggregation
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i * 10)
	}
	program := MustCompile("sum(arr) + avg(arr) + len(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_StringManipulation(b *testing.B) {
	program := MustCompile(`
		upper(trim(name)) + " - " + string(len(description)) + " chars"
	`)
	env := map[string]any{
		"name":        "  product name  ",
		"description": "This is a long product description that needs to be measured.",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// COMPLEX REAL-WORLD SCENARIOS
// =============================================================================

func BenchmarkScenario_FilterMapReduce(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("reduce(map(filter(arr, x => x % 2 == 0), x => x * 2), (a, b) => a + b, 0)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_PipelineProcessing(b *testing.B) {
	users := make([]any, 50)
	for i := range users {
		users[i] = map[string]any{
			"name":   fmt.Sprintf("User%d", i),
			"age":    int64(18 + i%50),
			"active": i%3 != 0,
		}
	}
	program := MustCompile(`filter(users, u => u.active && u.age >= 21) | len`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_DestructureAndCompute(b *testing.B) {
	program := MustCompile(`
		let {width, height, depth = 1} = box;
		width * height * depth
	`)
	env := map[string]any{
		"box": map[string]any{"width": int64(10), "height": int64(20)},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_SpreadAndAggregate(b *testing.B) {
	program := MustCompile("sum([...arr1, ...arr2, ...arr3])")
	env := map[string]any{
		"arr1": []any{int64(1), int64(2), int64(3)},
		"arr2": []any{int64(4), int64(5), int64(6)},
		"arr3": []any{int64(7), int64(8), int64(9)},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_ComplexValidation(b *testing.B) {
	program := MustCompile(`
		let {email, password, age, country} = user;
		len(email) > 0 && contains(email, "@") &&
		len(password) >= 8 &&
		age >= 18 && age <= 120 &&
		country in ["US", "UK", "CA", "AU"]
	`)
	env := map[string]any{
		"user": map[string]any{
			"email":    "test@example.com",
			"password": "securepass123",
			"age":      int64(25),
			"country":  "US",
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_SalesAnalytics(b *testing.B) {
	// Complex scenario: group sales by region, calculate stats, sort by total
	sales := make([]any, 100)
	regions := []string{"East", "West", "North", "South"}
	for i := range sales {
		sales[i] = map[string]any{
			"id":     int64(i),
			"region": regions[i%len(regions)],
			"amount": int64((i*17)%1000 + 100),
			"rep":    fmt.Sprintf("Rep%d", i%10),
		}
	}
	program := MustCompile(`
		groupBy(sales, s => s.region) | map(g => {
			"region": g.key,
			"total": sum(g.items | pluck("amount")),
			"count": len(g.items),
			"avg": round(avg(g.items | pluck("amount")))
		})
	`)
	env := map[string]any{"sales": sales}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_LeaderboardRanking(b *testing.B) {
	// Sort players by score descending, then take top 10
	players := make([]any, 100)
	for i := range players {
		players[i] = map[string]any{
			"name":  fmt.Sprintf("Player%d", i),
			"score": int64((i*31)%10000 + 1000),
			"level": int64(i%50 + 1),
		}
	}
	program := MustCompile(`
		sortBy(players, p => p.score, true) | take(10) | map(p => {
			"name": p.name,
			"score": p.score,
			"badge": p.score > 9000 ? "gold" : p.score > 5000 ? "silver" : "bronze"
		})
	`)
	env := map[string]any{"players": players}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_InventoryReport(b *testing.B) {
	// Group products by category, find low stock items, sort by urgency
	products := make([]any, 100)
	categories := []string{"Electronics", "Clothing", "Food", "Books", "Toys"}
	for i := range products {
		products[i] = map[string]any{
			"sku":          fmt.Sprintf("SKU%04d", i),
			"name":         fmt.Sprintf("Product%d", i),
			"category":     categories[i%len(categories)],
			"stock":        int64(i % 50),
			"reorderLevel": int64(20),
		}
	}
	program := MustCompile(`
		let lowStock = products | filter(p => p.stock <= p.reorderLevel);
		groupBy(lowStock, p => p.category) | map(g => {
			"category": g.key,
			"count": len(g.items),
			"items": sortBy(g.items, i => i.stock) | take(3) | pluck("name")
		})
	`)
	env := map[string]any{"products": products}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_UserSegmentation(b *testing.B) {
	// Segment users by activity, sort each segment, calculate metrics
	users := make([]any, 100)
	for i := range users {
		users[i] = map[string]any{
			"id":            int64(i),
			"name":          fmt.Sprintf("User%d", i),
			"lastLogin":     int64(i % 30),
			"purchaseCount": int64(i % 20),
			"totalSpent":    int64((i*37)%5000 + 100),
		}
	}
	program := MustCompile(`
		let segments = [
			{"name": "VIP", "filter": u => u.totalSpent > 3000},
			{"name": "Active", "filter": u => u.lastLogin < 7 && u.totalSpent <= 3000},
			{"name": "AtRisk", "filter": u => u.lastLogin >= 7 && u.lastLogin < 30},
			{"name": "Churned", "filter": u => u.lastLogin >= 30}
		];
		let vip = users | filter(u => u.totalSpent > 3000) | sortBy(u => u.totalSpent, true);
		{
			"vipCount": len(vip),
			"vipRevenue": sum(vip | pluck("totalSpent")),
			"topVIP": first(vip)?.name ?? "None"
		}
	`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}
