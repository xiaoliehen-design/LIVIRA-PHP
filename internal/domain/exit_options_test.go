package domain

import "testing"

func TestExitOptionsMatchInventoryTypeMatrix(t *testing.T) {
	tests := map[InventoryType]map[string]bool{
		InventoryBTD: {
			"impor_untuk_dipakai": true, "reekspor": true, "batal_ekspor": true, "ekspor": true,
			"keluarkan_ke_tpb": true, "lelang": true, "musnah": true, "psp": true, "hibah": true,
			"diserahkan_ke_aph_lain": true,
		},
		InventoryBDN: {
			"pembatalan_bdn": true, "reekspor": true, "lelang": true, "musnah": true, "psp": true,
			"hibah": true, "diserahkan_ke_ppns": true, "diserahkan_ke_aph_lain": true,
		},
		InventoryBMMN: {
			"lelang": true, "musnah": true, "psp": true, "hibah": true, "penghapusan": true,
		},
		InventoryTitipan: {
			"pengeluaran_barang_titipan": true,
		},
	}

	allCodes := []string{
		"impor_untuk_dipakai", "reekspor", "batal_ekspor", "ekspor", "keluarkan_ke_tpb",
		"lelang", "musnah", "psp", "hibah", "diserahkan_ke_aph_lain", "pembatalan_bdn",
		"diserahkan_ke_ppns", "penghapusan", "pengeluaran_barang_titipan",
	}
	for kind, allowed := range tests {
		for _, code := range allCodes {
			if got := ValidExitType(kind, code); got != allowed[code] {
				t.Errorf("ValidExitType(%s, %s)=%v, want %v", kind, code, got, allowed[code])
			}
		}
	}
}
