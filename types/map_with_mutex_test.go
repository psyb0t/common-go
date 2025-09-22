package commontypes

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewMapWithMutex(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates new empty map"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMapWithMutex[string, int]()
			assert.NotNil(t, m)
			assert.Equal(t, 0, m.Len())
		})
	}
}

func TestMapWithMutex_Set_Get(t *testing.T) {
	tests := []struct {
		name       string
		operations []struct {
			key   string
			value int
		}
		getKey      string
		expectValue int
		expectFound bool
	}{
		{
			name: "set and get existing key",
			operations: []struct {
				key   string
				value int
			}{{"key1", 100}},
			getKey:      "key1",
			expectValue: 100,
			expectFound: true,
		},
		{
			name: "get non-existent key",
			operations: []struct {
				key   string
				value int
			}{{"key1", 100}},
			getKey:      "nonexistent",
			expectValue: 0,
			expectFound: false,
		},
		{
			name: "overwrite existing key",
			operations: []struct {
				key   string
				value int
			}{{"key1", 100}, {"key1", 200}},
			getKey:      "key1",
			expectValue: 200,
			expectFound: true,
		},
		{
			name: "multiple keys",
			operations: []struct {
				key   string
				value int
			}{{"key1", 100}, {"key2", 200}},
			getKey:      "key2",
			expectValue: 200,
			expectFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMapWithMutex[string, int]()

			for _, op := range tt.operations {
				m.Set(op.key, op.value)
			}

			value, exists := m.Get(tt.getKey)
			assert.Equal(t, tt.expectFound, exists)
			if tt.expectFound {
				assert.Equal(t, tt.expectValue, value)
			}
		})
	}
}

func TestMapWithMutex_Delete(t *testing.T) {
	tests := []struct {
		name            string
		initialData     map[string]int
		deleteKey       string
		expectValue     int
		expectFound     bool
		expectRemaining int
	}{
		{
			name:            "delete existing key",
			initialData:     map[string]int{"key1": 100, "key2": 200},
			deleteKey:       "key1",
			expectValue:     100,
			expectFound:     true,
			expectRemaining: 1,
		},
		{
			name:            "delete non-existent key",
			initialData:     map[string]int{"key1": 100},
			deleteKey:       "nonexistent",
			expectValue:     0,
			expectFound:     false,
			expectRemaining: 1,
		},
		{
			name:            "delete from empty map",
			initialData:     map[string]int{},
			deleteKey:       "key1",
			expectValue:     0,
			expectFound:     false,
			expectRemaining: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMapWithMutex[string, int]()

			for key, value := range tt.initialData {
				m.Set(key, value)
			}

			deletedValue, exists := m.Delete(tt.deleteKey)
			assert.Equal(t, tt.expectFound, exists)
			assert.Equal(t, tt.expectValue, deletedValue)
			assert.Equal(t, tt.expectRemaining, m.Len())

			if tt.expectFound {
				_, stillExists := m.Get(tt.deleteKey)
				assert.False(t, stillExists)
			}
		})
	}
}

func TestMapWithMutex_Clear(t *testing.T) {
	tests := []struct {
		name        string
		initialData map[string]int
		expectLen   int
	}{
		{
			name:        "clear non-empty map",
			initialData: map[string]int{"key1": 100, "key2": 200, "key3": 300},
			expectLen:   3,
		},
		{
			name:        "clear empty map",
			initialData: map[string]int{},
			expectLen:   0,
		},
		{
			name:        "clear single item map",
			initialData: map[string]int{"key1": 100},
			expectLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMapWithMutex[string, int]()

			for key, value := range tt.initialData {
				m.Set(key, value)
			}

			old := m.Clear()
			assert.Equal(t, tt.expectLen, len(old))
			assert.Equal(t, 0, m.Len())

			for key, expectedValue := range tt.initialData {
				assert.Equal(t, expectedValue, old[key])
				_, exists := m.Get(key)
				assert.False(t, exists)
			}
		})
	}
}

func TestMapWithMutex_Len(t *testing.T) {
	tests := []struct {
		name       string
		operations []func(*MapWithMutex[string, int])
		expectLen  int
	}{
		{
			name:       "empty map",
			operations: []func(*MapWithMutex[string, int]){},
			expectLen:  0,
		},
		{
			name: "single item",
			operations: []func(*MapWithMutex[string, int]){
				func(m *MapWithMutex[string, int]) { m.Set("key1", 100) },
			},
			expectLen: 1,
		},
		{
			name: "multiple items",
			operations: []func(*MapWithMutex[string, int]){
				func(m *MapWithMutex[string, int]) { m.Set("key1", 100) },
				func(m *MapWithMutex[string, int]) { m.Set("key2", 200) },
				func(m *MapWithMutex[string, int]) { m.Set("key3", 300) },
			},
			expectLen: 3,
		},
		{
			name: "set and delete",
			operations: []func(*MapWithMutex[string, int]){
				func(m *MapWithMutex[string, int]) { m.Set("key1", 100) },
				func(m *MapWithMutex[string, int]) { m.Set("key2", 200) },
				func(m *MapWithMutex[string, int]) { m.Delete("key1") },
			},
			expectLen: 1,
		},
		{
			name: "clear all",
			operations: []func(*MapWithMutex[string, int]){
				func(m *MapWithMutex[string, int]) { m.Set("key1", 100) },
				func(m *MapWithMutex[string, int]) { m.Set("key2", 200) },
				func(m *MapWithMutex[string, int]) { m.Clear() },
			},
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMapWithMutex[string, int]()

			for _, op := range tt.operations {
				op(m)
			}

			assert.Equal(t, tt.expectLen, m.Len())
		})
	}
}

func TestMapWithMutex_GenericTypes(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "uuid to string mapping",
			test: func(t *testing.T) {
				m := NewMapWithMutex[uuid.UUID, string]()
				id1 := uuid.New()
				id2 := uuid.New()

				m.Set(id1, "value1")
				m.Set(id2, "value2")

				value, exists := m.Get(id1)
				assert.True(t, exists)
				assert.Equal(t, "value1", value)

				value, exists = m.Get(id2)
				assert.True(t, exists)
				assert.Equal(t, "value2", value)
			},
		},
		{
			name: "int to pointer mapping",
			test: func(t *testing.T) {
				type TestStruct struct {
					Name  string
					Value int
				}

				m := NewMapWithMutex[int, *TestStruct]()
				obj1 := &TestStruct{Name: "test1", Value: 100}
				obj2 := &TestStruct{Name: "test2", Value: 200}

				m.Set(1, obj1)
				m.Set(2, obj2)

				retrieved, exists := m.Get(1)
				assert.True(t, exists)
				assert.Equal(t, obj1, retrieved)
				assert.Equal(t, "test1", retrieved.Name)
				assert.Equal(t, 100, retrieved.Value)

				deleted, exists := m.Delete(2)
				assert.True(t, exists)
				assert.Equal(t, obj2, deleted)
			},
		},
		{
			name: "string to interface mapping",
			test: func(t *testing.T) {
				m := NewMapWithMutex[string, any]()

				m.Set("int", 42)
				m.Set("string", "hello")
				m.Set("slice", []int{1, 2, 3})

				intVal, exists := m.Get("int")
				assert.True(t, exists)
				assert.Equal(t, 42, intVal)

				strVal, exists := m.Get("string")
				assert.True(t, exists)
				assert.Equal(t, "hello", strVal)

				sliceVal, exists := m.Get("slice")
				assert.True(t, exists)
				assert.Equal(t, []int{1, 2, 3}, sliceVal)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}
}

func TestMapWithMutex_ConcurrentAccess(t *testing.T) {
	tests := []struct {
		name           string
		numGoroutines  int
		numOperations  int
		deleteEveryNth int
	}{
		{
			name:           "low concurrency",
			numGoroutines:  10,
			numOperations:  100,
			deleteEveryNth: 2,
		},
		{
			name:           "high concurrency",
			numGoroutines:  100,
			numOperations:  1000,
			deleteEveryNth: 2,
		},
		{
			name:           "mostly reads",
			numGoroutines:  50,
			numOperations:  500,
			deleteEveryNth: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMapWithMutex[int, string]()
			var wg sync.WaitGroup

			for i := 0; i < tt.numGoroutines; i++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()
					for j := 0; j < tt.numOperations; j++ {
						key := goroutineID*tt.numOperations + j
						value := "value" + string(rune(key))

						m.Set(key, value)

						if retrievedValue, exists := m.Get(key); exists {
							assert.Equal(t, value, retrievedValue)
						}

						if j%tt.deleteEveryNth == 0 {
							m.Delete(key)
						}
					}
				}(i)
			}

			wg.Wait()

			finalLen := m.Len()
			assert.GreaterOrEqual(t, finalLen, 0)
		})
	}
}
