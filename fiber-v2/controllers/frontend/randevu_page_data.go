package frontend

import (
	"log"
	"models"
	"strings"

	"lib"
)

// P3-1 — /randevu talep sihirbazının okuduğu veri. Yalnız public, aktif
// kayıtlar ve hastanın seçim yapması için gereken alanlar okunur (TC, e-posta,
// ücret vb. okunmaz). Hata olursa ilgili liste boş döner; sayfa yine açılır,
// ziyaretçi "Emin değilim" / "Fark etmez" ile devam edebilir.

type RandevuBolum struct {
	Brid string
	Name string
}

type RandevuDoktor struct {
	Drid     string
	Brid     string
	FullName string
}

type RandevuMerkez struct {
	Sid       string
	Name      string
	City      string
	District  string
	Address   string
	Phone     string
	PhoneHref string
	Bolumler  []RandevuBolum
	Doktorlar []RandevuDoktor
}

// telHref telefon metnini tel: bağlantısı için yalnız rakam ve baştaki + olacak şekilde sadeleştirir.
func telHref(phone string) string {
	var b strings.Builder
	for i, r := range strings.TrimSpace(phone) {
		if (r >= '0' && r <= '9') || (r == '+' && i == 0) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func readRows(stage string, run func() ([]map[string]interface{}, error)) []map[string]interface{} {
	rows, err := run()
	if err != nil {
		log.Printf("operation=RandevuPage stage=%s", stage)
		return nil
	}
	return rows
}

func loadRandevuMerkezleri(utilities *models.Utilities) []RandevuMerkez {
	Orm := utilities.Orm

	subeRows := readRows("subeler_read", func() ([]map[string]interface{}, error) {
		q := Orm.Select([]string{"sid", "name", "city", "district", "address", "phone"})
		q.Table("subeler")
		q.Where("is_active", "=", true)
		q.OrderBy("name", "ASC")
		q.Finish()
		if err := q.Execute(); err != nil {
			return nil, err
		}
		return q.Rows()
	})

	bolumRows := readRows("branslar_read", func() ([]map[string]interface{}, error) {
		q := Orm.Select([]string{"brid", "name", "sid"})
		q.Table("branslar")
		q.Where("is_active", "=", true)
		q.OrderBy("name", "ASC")
		q.Finish()
		if err := q.Execute(); err != nil {
			return nil, err
		}
		return q.Rows()
	})

	doktorRows := readRows("doktorlar_read", func() ([]map[string]interface{}, error) {
		q := Orm.Select([]string{"drid", "title", "first_name", "last_name", "sid", "brid"})
		q.Table("doktorlar")
		q.Where("is_active", "=", true)
		q.And("online_appointment", "=", true)
		q.OrderBy("first_name", "ASC")
		q.Finish()
		if err := q.Execute(); err != nil {
			return nil, err
		}
		return q.Rows()
	})

	merkezler := make([]RandevuMerkez, 0, len(subeRows))
	index := map[string]int{}
	for _, row := range subeRows {
		phone := strings.TrimSpace(lib.String(row["phone"]))
		m := RandevuMerkez{
			Sid:       lib.String(row["sid"]),
			Name:      lib.String(row["name"]),
			City:      lib.String(row["city"]),
			District:  lib.String(row["district"]),
			Address:   lib.String(row["address"]),
			Phone:     phone,
			PhoneHref: telHref(phone),
		}
		index[m.Sid] = len(merkezler)
		merkezler = append(merkezler, m)
	}

	for _, row := range bolumRows {
		if i, ok := index[lib.String(row["sid"])]; ok {
			merkezler[i].Bolumler = append(merkezler[i].Bolumler, RandevuBolum{
				Brid: lib.String(row["brid"]),
				Name: lib.String(row["name"]),
			})
		}
	}

	for _, row := range doktorRows {
		i, ok := index[lib.String(row["sid"])]
		if !ok {
			continue
		}
		name := strings.TrimSpace(strings.Join([]string{
			lib.String(row["title"]), lib.String(row["first_name"]), lib.String(row["last_name"]),
		}, " "))
		merkezler[i].Doktorlar = append(merkezler[i].Doktorlar, RandevuDoktor{
			Drid:     lib.String(row["drid"]),
			Brid:     lib.String(row["brid"]),
			FullName: name,
		})
	}

	return merkezler
}
