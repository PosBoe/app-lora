package device

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bom = "\xef\xbb\xbf"

func TestParseCSV(t *testing.T) {
	t.Run("comma file", func(t *testing.T) {
		rows, err := parseCSV([]byte("dev_eui,name,vendor,model,type,address,note,groups\n" +
			"70:B3:D5:7E:D0:00:12:34,Bloc A,Milesight,EM300-DI,apă,\"str. Mare 1, ap. 2\",,\"Botanica;Sector 1\"\n"))
		require.NoError(t, err)
		require.Len(t, rows, 1)
		r := rows[0]
		assert.Nil(t, r.Err)
		assert.Equal(t, 2, r.Line)
		assert.Equal(t, "70b3d57ed0001234", r.Fields.DevEUI)
		assert.Equal(t, "water", r.Fields.MeterType)
		assert.Equal(t, "str. Mare 1, ap. 2", r.Fields.Address)
		assert.Equal(t, "Milesight", r.Vendor)
		assert.Equal(t, []string{"Botanica", "Sector 1"}, r.Groups)
	})

	t.Run("Excel file: BOM, semicolons, header in any order and case, Cyrillic", func(t *testing.T) {
		rows, err := parseCSV([]byte(bom + "Name;DEV_EUI;Model;Vendor;Type;Groups\r\n" +
			"Котельная;70b3d57ed0001235;CMi4110;Elvaco;газ;\"Север;север\"\r\n"))
		require.NoError(t, err)
		require.Len(t, rows, 1)
		r := rows[0]
		assert.Nil(t, r.Err)
		assert.Equal(t, "Котельная", r.Fields.Name)
		assert.Equal(t, "gas", r.Fields.MeterType)
		assert.Equal(t, []string{"Север"}, r.Groups, "group names repeat in any case only once")
	})

	t.Run("row errors keep the other rows", func(t *testing.T) {
		rows, err := parseCSV([]byte("dev_eui,name,vendor,model,type\n" +
			"12,a,V,M,\n" +
			"70b3d57ed0001236,,V,M,\n" +
			"70b3d57ed0001237,c,,M,\n" +
			"70b3d57ed0001238,d,V,M,steam\n" +
			"70b3d57ed0001239,e,V,M,\n" +
			"\n" +
			"70:b3:d5:7e:d0:00:12:39,f,V,M,\n"))
		require.NoError(t, err)
		codes := make([]string, len(rows))
		for i, r := range rows {
			if r.Err != nil {
				codes[i] = r.Err.code
			}
		}
		assert.Equal(t, []string{
			"dev_eui_invalid", "device_name_required", "vendor_required", "meter_type_invalid",
			"", "duplicate_in_file",
		}, codes, "a blank line is skipped")
		assert.Equal(t, 8, rows[5].Line)
	})

	t.Run("file errors", func(t *testing.T) {
		cases := []struct {
			name string
			data string
			want *ruleError
		}{
			{"empty", "", errImportEmpty},
			{"not UTF-8", "dev_eui,name,vendor,model\n70b3d57ed0001234,Ap\xe3,V,M\n", errImportNotUTF8},
			{"missing column", "dev_eui,name,vendor\n", errImportMissingColumn},
			{"unknown column", "dev_eui,name,vendor,model,colour\n", errImportUnknownColumn},
			{"repeated column", "dev_eui,name,vendor,model,name\n", errImportUnknownColumn},
			{"broken quotes", "dev_eui,name,vendor,model\n\"70b3,a,V,M\n", errImportMalformed},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := parseCSV([]byte(tc.data))
				assert.ErrorIs(t, err, tc.want)
			})
		}
	})

	t.Run("row limit", func(t *testing.T) {
		var b strings.Builder
		b.WriteString("dev_eui,name,vendor,model\n")
		for i := range maxImportRows + 1 {
			fmt.Fprintf(&b, "%016x,d%d,V,M\n", i+1, i)
		}
		_, err := parseCSV([]byte(b.String()))
		assert.ErrorIs(t, err, errImportTooManyRows)
	})
}
