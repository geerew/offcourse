package pagination

import (
	"encoding/json"
	"testing"

	"github.com/geerew/off-course/utils/types"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully parsing a page value
func Test_ParsePage(t *testing.T) {
	var tests = []struct {
		in       string
		expected int
	}{
		{"1", 1},
		{"", 1},
		{"abc", 1},
		{"-1", 1},
		{"0", 1},
		{"5", 5},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expected, ParsePage(tt.in), "input %q", tt.in)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully parsing a perPage value
func Test_ParsePerPage(t *testing.T) {
	var tests = []struct {
		in       string
		expected int
	}{
		{"1", 1},
		{"", DefaultPerPage},
		{"abc", DefaultPerPage},
		{"-1", DefaultPerPage},
		{"0", DefaultPerPage},
		{"5", 5},
		{"99999", MaxPerPage},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expected, ParsePerPage(tt.in), "input %q", tt.in)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_NewFromQuery(t *testing.T) {
	// Test successfully creating a pagination with default values
	t.Run("defaults", func(t *testing.T) {
		p := New(ParsePage(""), ParsePerPage(""))
		p.SetCount(1)

		require.Equal(t, 1, p.page)
		require.Equal(t, DefaultPerPage, p.perPage)
		require.Equal(t, 1, p.TotalItems())
		require.Equal(t, 1, p.TotalPages())
	})

	// Test successfully creating a pagination with values
	t.Run("values", func(t *testing.T) {
		p := New(ParsePage("2"), ParsePerPage("10"))
		p.SetCount(24)

		require.Equal(t, 2, p.page)
		require.Equal(t, 24, p.TotalItems())
		require.Equal(t, 3, p.TotalPages())
	})

	// Test error when invalid values are provided
	t.Run("invalid values", func(t *testing.T) {
		p := New(ParsePage("-20"), ParsePerPage("bob"))
		p.SetCount(24)

		require.Equal(t, 1, p.page)
		require.Equal(t, DefaultPerPage, p.perPage)
		require.Equal(t, 24, p.TotalItems())
		require.Equal(t, 1, p.TotalPages())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_New(t *testing.T) {
	// Test successfully creating a pagination with default values
	t.Run("no values", func(t *testing.T) {
		p := New(1, DefaultPerPage)
		p.SetCount(1)

		require.Equal(t, 1, p.page)
		require.Equal(t, DefaultPerPage, p.perPage)
		require.Equal(t, 1, p.TotalItems())
	})

	// Test successfully creating a pagination with values
	t.Run("values", func(t *testing.T) {
		p := New(2, 10)
		p.SetCount(24)

		require.Equal(t, 2, p.page)
		require.Equal(t, 24, p.TotalItems())
		require.Equal(t, 3, p.TotalPages())
	})

	// Test successfully creating a pagination when the perPage value is above
	// the max
	t.Run("above max", func(t *testing.T) {
		p := New(1, MaxPerPage+1)
		p.SetCount(1)

		require.Equal(t, 1, p.page)
		require.Equal(t, MaxPerPage, p.perPage)
	})

	// Test successfully normalizing invalid values
	t.Run("invalid values", func(t *testing.T) {
		p := New(-1, -1)
		p.SetCount(24)

		require.Equal(t, 1, p.page)
		require.Equal(t, DefaultPerPage, p.perPage)
		require.Equal(t, 24, p.TotalItems())
		require.Equal(t, 1, p.TotalPages())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the limit value
func Test_Limit(t *testing.T) {
	var tests = []struct {
		perPage  string
		expected int
	}{
		{"1", 1},
		{"", DefaultPerPage},
		{"abc", DefaultPerPage},
		{"-1", DefaultPerPage},
		{"0", DefaultPerPage},
		{"5", 5},
	}

	for _, tt := range tests {
		p := New(1, ParsePerPage(tt.perPage))
		require.Equal(t, tt.expected, p.Limit(), "perPage %q", tt.perPage)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the offset value
func Test_Offset(t *testing.T) {
	var tests = []struct {
		page     string
		perPage  string
		expected int
	}{
		{"", "", 0},
		{"abc", "def", 0},
		{"-1", "40", 0},
		{"0", "10", 0},
		{"1", "10", 0},
		{"2", "10", 10},
		{"5", "10", 40},
		{"20", "30", 570},
	}

	for _, tt := range tests {
		p := New(ParsePage(tt.page), ParsePerPage(tt.perPage))
		require.Equal(t, tt.expected, p.Offset(), "page=%q perPage=%q", tt.page, tt.perPage)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_BuildResult(t *testing.T) {
	// Test successfully building a result object
	t.Run("success", func(t *testing.T) {
		p := New(1, DefaultPerPage)
		p.SetCount(24)

		type Data struct {
			ID        string         `json:"id"`
			CreatedAt types.DateTime `json:"createdAt"`
		}

		data := []Data{
			{ID: "1", CreatedAt: types.NowDateTime()},
			{ID: "2", CreatedAt: types.NowDateTime()},
		}

		result, err := p.BuildResult(data)
		require.NoError(t, err)
		require.Len(t, result.Items, 2)

		for i, raw := range result.Items {
			var d Data
			require.Nil(t, json.Unmarshal(raw, &d))
			require.Equal(t, data[i].ID, d.ID)
			require.Equal(t, data[i].CreatedAt.String(), d.CreatedAt.String())
		}
	})

	// Test error when the input is not a slice
	t.Run("invalid data", func(t *testing.T) {
		p := New(1, DefaultPerPage)
		p.SetCount(24)

		result, err := p.BuildResult("data")
		require.EqualError(t, err, "input is not a slice")
		require.Nil(t, result)
	})

	// Test error when marshalling invalid data
	t.Run("error marshalling", func(t *testing.T) {
		p := New(1, DefaultPerPage)
		p.SetCount(24)

		badData := []struct {
			UnsupportedField chan int `json:"unsupportedField"`
		}{
			{UnsupportedField: make(chan int)},
		}

		result, err := p.BuildResult(badData)
		require.EqualError(t, err, "json: unsupported type: chan int")
		require.Nil(t, result)
	})
}
