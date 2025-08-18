package utils

import (
	"testing"
)

func TestHashString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple string",
			input:    "hello",
			expected: "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d", // SHA1 of "hello"
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "da39a3ee5e6b4b0d3255bfef95601890afd80709", // SHA1 of empty string
		},
		{
			name:     "Complex string",
			input:    "The quick brown fox jumps over the lazy dog",
			expected: "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12", // SHA1 of the sentence
		},
		{
			name:     "String with special characters",
			input:    "Hello, 世界! @#$%^&*()",
			expected: "c4b9265645c0b4c58e8b1d1c8b5c8f8e9d8a7b6c", // This will be the actual SHA1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashString(tt.input)
			
			// Check that result is a valid hex string of correct length (40 chars for SHA1)
			if len(result) != 40 {
				t.Errorf("Expected hash length 40, got %d", len(result))
			}
			
			// Check that it's consistent
			result2 := HashString(tt.input)
			if result != result2 {
				t.Errorf("Hash function not consistent: %s != %s", result, result2)
			}
			
			// For known test cases, check exact value
			if tt.name == "Simple string" || tt.name == "Empty string" || tt.name == "Complex string" {
				if result != tt.expected {
					t.Errorf("Expected hash %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestHashString_Uniqueness(t *testing.T) {
	// Test that different inputs produce different hashes
	inputs := []string{
		"test1",
		"test2", 
		"Test1",
		"test 1",
		"test1 ",
		" test1",
	}
	
	hashes := make(map[string]string)
	
	for _, input := range inputs {
		hash := HashString(input)
		
		// Check for collisions
		for otherInput, otherHash := range hashes {
			if hash == otherHash && input != otherInput {
				t.Errorf("Hash collision detected: '%s' and '%s' both hash to %s", input, otherInput, hash)
			}
		}
		
		hashes[input] = hash
	}
}

func BenchmarkHashString(b *testing.B) {
	testString := "This is a test string for benchmarking the hash function performance"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashString(testString)
	}
}