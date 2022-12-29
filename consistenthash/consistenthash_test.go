package consistenthash

import (
    "strconv"
    "testing"
)

func TestConsistentHash(t *testing.T) {
    numHash := func(data []byte) uint32 {
        h, err := strconv.Atoi(string(data))
        if err != nil {
            t.Fatalf("data bytes must be appropriate integer")
        }
        return uint32(h)
    }

    testCases := map[string]string{
        "10": "1", "11": "1", "12": "1",
        "20": "2", "21": "2", "22": "2",
        "30": "3", "31": "3", "32": "3",

        "5": "1", "15": "2", "25": "3", "35": "1",
    }

    ch := New(3, numHash)
    ch.Add("1") // virtual nodes {"11", 11}, {"12", 12}, {"13", 13}
    ch.Add("2") // virtual nodes {"21", 21}, {"22", 22}, {"23", 23}
    ch.Add("3") // virtual nodes {"31", 31}, {"32", 32}, {"33", 33}

    for k, v := range testCases {
        physical, ok := ch.Get(k)
        if !ok {
            t.Fatalf("Consistent hash is empty")
        }

        if physical != v {
            t.Fatalf("virtual node %s should be mapped to %s, but now %s", k, v, physical)
        }
    }

    ch.Remove("1")
    ch.Remove("2")
    for k, _ := range testCases {
        physical, ok := ch.Get(k)
        if !ok {
            t.Fatalf("Consistent hash is empty")
        }

        if physical != "3" {
            t.Fatalf("virtual node %s should be mapped to 3, but now %s", k, physical)
        }
    }
}
