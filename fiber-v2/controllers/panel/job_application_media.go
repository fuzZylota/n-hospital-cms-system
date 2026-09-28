package panel

import (
	"lib"
	"models"
	"os"

	"github.com/gofiber/fiber/v2"
)

func JobApplicationMedia(utilities *models.Utilities) fiber.Handler {
	lookup := func(jaid string, mid int64) (lib.JobApplicationMediaRecord, error) {
		if utilities == nil || utilities.Orm == nil {
			return lib.JobApplicationMediaRecord{}, os.ErrInvalid
		}
		application := utilities.Orm.Select([]string{"jaid", "cv_file_mid", "diploma_file_mid"})
		application.Table("job_applications")
		application.Where("jaid", "=", jaid)
		application.Finish()
		if err := application.Execute(); err != nil {
			return lib.JobApplicationMediaRecord{}, err
		}
		appRows, err := application.Rows()
		if err != nil {
			return lib.JobApplicationMediaRecord{}, err
		}
		if len(appRows) != 1 {
			return lib.JobApplicationMediaRecord{}, os.ErrNotExist
		}

		media := utilities.Orm.Select([]string{"mid", "target_id", "file_type", "file_path"})
		media.Table("medias")
		media.Where("mid", "=", mid)
		media.Finish()
		if err := media.Execute(); err != nil {
			return lib.JobApplicationMediaRecord{}, err
		}
		mediaRows, err := media.Rows()
		if err != nil {
			return lib.JobApplicationMediaRecord{}, err
		}
		if len(mediaRows) != 1 {
			return lib.JobApplicationMediaRecord{}, os.ErrNotExist
		}

		return lib.JobApplicationMediaRecord{
			Jaid:       lib.String(appRows[0]["jaid"]),
			CVMid:      lib.Int64(appRows[0]["cv_file_mid"]),
			DiplomaMid: lib.Int64(appRows[0]["diploma_file_mid"]),
			Mid:        lib.Int64(mediaRows[0]["mid"]),
			TargetID:   lib.String(mediaRows[0]["target_id"]),
			FileType:   lib.String(mediaRows[0]["file_type"]),
			FilePath:   lib.String(mediaRows[0]["file_path"]),
		}, nil
	}
	return lib.JobApplicationMediaHandler(os.Getenv("ROOT_DIRECTORY"), lookup)
}
