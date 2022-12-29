package lru

import (
    "fmt"
    "testing"
)

type simpleStruct struct {
    int
    string
}

type complexStruct struct {
    int
    simpleStruct
}

type testCase struct {
    name       string
    keyToAdd   any
    keyToGet   any
    expectedOk bool
}

var getTests = []testCase{
    {"string_hit", "myKey", "myKey", true},
    {"string_miss", "myKey", "nonsense", false},
    {"simple_struct_hit", simpleStruct{1, "two"}, simpleStruct{1, "two"}, true},
    {"simple_struct_miss", simpleStruct{1, "two"}, simpleStruct{0, "noway"}, false},
    {"complex_struct_hit", complexStruct{1, simpleStruct{2, "three"}},
        complexStruct{1, simpleStruct{2, "three"}}, true},
}

func TestGet(t *testing.T) {
    for _, tc := range getTests {
        switch tc.keyToAdd.(type) {
        case string:
            testGetGeneric[string](t, tc)
        case simpleStruct:
            testGetGeneric[simpleStruct](t, tc)
        case complexStruct:
            testGetGeneric[complexStruct](t, tc)
        }
    }
}

func testGetGeneric[K comparable](t *testing.T, tc testCase) {
    lru := New[K](0, nil)
    lru.Add(tc.keyToAdd.(K), 1234)
    val, ok := lru.Get(tc.keyToGet.(K))
    if ok != tc.expectedOk {
        t.Fatalf("%s: cache hit = %v; want %v", tc.name, ok, !ok)
    } else if ok && val != 1234 {
        t.Fatalf("%s expected get to return 1234 but got %v", tc.name, val)
    }
}

func TestRemove(t *testing.T) {
    lru := New[string](0, nil)
    lru.Add("myKey", 1234)
    if val, ok := lru.Get("myKey"); !ok {
        t.Fatal("TestRemove returned no match")
    } else if val != 1234 {
        t.Fatalf("TestRemove failed.  Expected %d, got %v", 1234, val)
    }

    lru.Remove("myKey")
    if _, ok := lru.Get("myKey"); ok {
        t.Fatal("TestRemove returned a removed entry")
    }
}

func TestEvict(t *testing.T) {
    evictedKeys := make([]string, 0)
    onEvictedFun := func(key string, value any) {
        evictedKeys = append(evictedKeys, key)
    }

    lru := New(20, onEvictedFun)
    for i := 0; i < 22; i++ {
        lru.Add(fmt.Sprintf("myKey%d", i), 1234)
    }

    if len(evictedKeys) != 2 {
        t.Fatalf("got %d evicted keys; want 2", len(evictedKeys))
    }
    if evictedKeys[0] != "myKey0" {
        t.Fatalf("got %v in first evicted key; want %s", evictedKeys[0], "myKey0")
    }
    if evictedKeys[1] != "myKey1" {
        t.Fatalf("got %v in second evicted key; want %s", evictedKeys[1], "myKey1")
    }
}
