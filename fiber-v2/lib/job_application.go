package lib

// JobApplicationUploadRootAvailable is checked before inserting an application.
// CV upload remains optional; notification attachments must not block a record.
func JobApplicationUploadRootAvailable(hasCV bool, root string) bool {
	return !hasCV || root != ""
}
