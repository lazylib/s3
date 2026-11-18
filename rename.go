package s3

import (
	"context"
	"fmt"

	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Rename переименовывает объект oldname в newname внутри одного бакета.
// В S3 нет операции переименования, поэтому объект сначала копируется
// на стороне S3 (CopyObject, без передачи данных через приложение),
// а затем оригинал удаляется (DeleteObject).
func (c conf) Rename(oldname, newname string) error {
	// переименование "в самого себя" — ничего не делаем,
	// иначе последовательность copy+delete уничтожила бы объект
	if oldname == newname {
		return nil
	}

	if err := c.Copy(oldname, newname); err != nil {
		return err
	}

	if _, err := c.client.DeleteObject(context.TODO(), &s3Lib.DeleteObjectInput{
		Bucket: &c.settings.Bucket,
		Key:    &oldname,
	}); err != nil {
		return fmt.Errorf("объект %q скопирован в %q, но не удалось удалить оригинал: %v", oldname, newname, err)
	}

	return nil
}
