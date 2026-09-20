package postgres

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const (
	contactRequestSnapshotType   = "ContactRequestWorkflowSnapshot"
	contactRequestSnapshotReader = "ContactRequestWorkflowSnapshotReader"
	contactRequestSnapshotMethod = "ReadContactRequestWorkflowSnapshot"
)

type contactRequestScannerConfig struct {
	dataImportPath       string
	repositoryImportPath string
	helperImportPath     string
	contractPath         string
	repositoryPath       string
	optionsPath          string
	addContactPath       string
	helperPath           string
}

type contactRequestSource struct {
	relativePath string
	source       []byte
}

type contactRequestParsedSource struct {
	relativePath string
	file         *ast.File
}

type contactRequestScanContext struct {
	config                    contactRequestScannerConfig
	sources                   []contactRequestParsedSource
	packageTypes              map[contactRequestPackageIdentity]map[string][]contactRequestPackageType
	snapshotType              *ast.TypeSpec
	readerType                *ast.TypeSpec
	optionsRepositoryType     *ast.TypeSpec
	repositoryImplementation  *ast.FuncDecl
	addContactRequestFunction *ast.FuncDecl
	allowedNodes              map[ast.Node]bool
}

type contactRequestPackageIdentity struct {
	path string
	name string
}

type contactRequestPackageType struct {
	source      contactRequestParsedSource
	declaration *ast.TypeSpec
}

type contactRequestPackageSymbol struct {
	packageIdentity contactRequestPackageIdentity
	name            string
}

type contactRequestOwnershipResolution struct {
	objects map[*ast.Object]bool
	symbols map[contactRequestPackageSymbol]bool
}

type contactRequestImportBindings struct {
	dataAliases       map[string]bool
	repositoryAliases map[string]bool
	dotData           bool
}

func TestContactRequestWorkflowSnapshotHasOnlyApprovedProductionWiring(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	count, ok := scanContactRequestWorkspace(root)
	if !ok {
		t.Fatal("production reference scan failed")
	}
	if count != 10 {
		t.Fatal("unexpected contact-request production consumer")
	}
	if !hasExactApprovedContactRequestWiring(root) || !hasExactApprovedContactRequestSymbolInventory(root) {
		t.Fatal("approved contact-request production wiring changed")
	}
	if !hasOnlyApprovedContactRequestHelperReferences(root) {
		t.Fatal("unexpected contact-request helper reference")
	}
}

func TestContactRequestProductionReferenceScannerFixtures(t *testing.T) {
	config := contactRequestScannerConfig{
		dataImportPath:       "models/data",
		repositoryImportPath: "database/postgres",
		helperImportPath:     "post/contactrequestsnapshot",
		contractPath:         "models/data/contact_request_workflow_snapshot.go",
		repositoryPath:       "database/postgres/contact_request_workflow_snapshot.go",
		optionsPath:          "database/postgres/options.go",
		addContactPath:       "controllers/post/post.go",
		helperPath:           "controllers/post/contactrequestsnapshot/decision.go",
	}
	positive := []contactRequestSource{
		{relativePath: "consumer/default.go", source: []byte(`package consumer
import "models/data"
var _ data.ContactRequestWorkflowSnapshot`)},
		{relativePath: "consumer/alias.go", source: []byte(`package consumer
import contract "models/data"
var _ contract.ContactRequestWorkflowSnapshot`)},
		{relativePath: "consumer/interface_default.go", source: []byte(`package consumer
import "models/data"
var _ data.ContactRequestWorkflowSnapshotReader`)},
		{relativePath: "consumer/interface_alias.go", source: []byte(`package consumer
import contract "models/data"
var _ contract.ContactRequestWorkflowSnapshotReader`)},
		{relativePath: "consumer/function_value.go", source: []byte(`package consumer
import contract "models/data"
func consume(reader contract.ContactRequestWorkflowSnapshotReader) {
	_ = reader.ReadContactRequestWorkflowSnapshot
}`)},
		{relativePath: "consumer/function_value_default.go", source: []byte(`package consumer
import "models/data"
func consume(reader data.ContactRequestWorkflowSnapshotReader) {
	_ = reader.ReadContactRequestWorkflowSnapshot
}`)},
		{relativePath: "consumer/dot.go", source: []byte(`package consumer
import . "models/data"
var _ ContactRequestWorkflowSnapshotReader`)},
		{relativePath: "consumer/repository_alias.go", source: []byte(`package consumer
import repository "database/postgres"
func consume(reader *repository.OptionsRepository) {
	reader.ReadContactRequestWorkflowSnapshot(nil)
}`)},
	}
	for _, fixture := range positive {
		count, ok := countContactRequestProductionReferences([]contactRequestSource{fixture}, config)
		want := 1
		if fixture.relativePath == "consumer/function_value.go" || fixture.relativePath == "consumer/function_value_default.go" {
			want = 2
		}
		if !ok || count != want {
			t.Fatal("production reference scanner missed a protected consumer")
		}
	}

	negative := []contactRequestSource{
		{relativePath: "models/data/local.go", source: []byte(`package data
func f() { ContactRequestWorkflowSnapshot := 1; _ = ContactRequestWorkflowSnapshot }`)},
		{relativePath: "models/data/parameter.go", source: []byte(`package data
func f(ContactRequestWorkflowSnapshot int) { _ = ContactRequestWorkflowSnapshot }`)},
		{relativePath: "models/data/type_declaration.go", source: []byte(`package data
type ContactRequestWorkflowSnapshot struct{}`)},
		{relativePath: "database/postgres/function_declaration.go", source: []byte(`package postgres
func ReadContactRequestWorkflowSnapshot() {}`)},
		{relativePath: "models/data/field.go", source: []byte(`package data
type holder struct { ContactRequestWorkflowSnapshot int }`)},
		{relativePath: "models/data/import_declaration.go", source: []byte(`package data
import ContactRequestWorkflowSnapshot "example.invalid/other/data"
var _ = 1`)},
		{relativePath: "consumer/other_import.go", source: []byte(`package consumer
import data "example.invalid/other/data"
var _ data.ContactRequestWorkflowSnapshot`)},
		{relativePath: "consumer/import_shadow.go", source: []byte(`package consumer
import contract "models/data"
func f(contract struct { ContactRequestWorkflowSnapshot int }) { _ = contract.ContactRequestWorkflowSnapshot }`)},
		{relativePath: "consumer/text.go", source: []byte(`package consumer
// data.ContactRequestWorkflowSnapshot
const text = "ReadContactRequestWorkflowSnapshot"`)},
		{relativePath: "models/data/contact_request_workflow_snapshot.go", source: []byte(`package data
type ContactRequestWorkflowSnapshot struct{}`)},
		{relativePath: "database/postgres/contact_request_workflow_snapshot.go", source: []byte(`package postgres
func (repository *OptionsRepository) ReadContactRequestWorkflowSnapshot() {}`)},
		{relativePath: "database/postgres/different_receiver.go", source: []byte(`package postgres
type otherRepository struct{}
func f(repository *otherRepository) { _ = repository.ReadContactRequestWorkflowSnapshot }`)},
	}
	for _, fixture := range negative {
		count, ok := countContactRequestProductionReferences([]contactRequestSource{fixture}, config)
		if !ok || count != 0 {
			t.Fatal("production reference scanner accepted a non-consumer")
		}
	}
}

func TestContactRequestSamePackageRepositoryOwnershipFixtures(t *testing.T) {
	positive := []string{
		`func use(r *OptionsRepository) { _ = r.ReadContactRequestWorkflowSnapshot }`,
		`func use(r *OptionsRepository) { r.ReadContactRequestWorkflowSnapshot(nil) }`,
		`func use(r OptionsRepository) { _ = r.ReadContactRequestWorkflowSnapshot }`,
	}
	for _, extra := range positive {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{repositoryConsumer: extra}) {
			t.Fatal("cannot create same-package ownership fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 1 {
			t.Fatal("same-package repository consumer was not owned by the canonical type")
		}
	}

	negative := []contactRequestFixtureOptions{
		{repositorySidecarConsumer: `type OptionsRepository struct{ ReadContactRequestWorkflowSnapshot int }
func use(r *OptionsRepository) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{libConsumer: `import fake "example.invalid/fake"
func use(r *fake.OptionsRepository) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{libConsumer: `func use(r struct{ ReadContactRequestWorkflowSnapshot int }) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{libConsumer: `func use(r interface{ ReadContactRequestWorkflowSnapshot() }) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositoryConsumer: `type OtherRepository struct{ ReadContactRequestWorkflowSnapshot int }
func use(r *OtherRepository) { _ = r.ReadContactRequestWorkflowSnapshot }`},
	}
	for _, options := range negative {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, options) {
			t.Fatal("cannot create repository ownership rejection fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 0 {
			t.Fatal("repository ownership scanner accepted a non-canonical receiver")
		}
	}
}

func TestContactRequestOptionsRepositoryAliasOwnershipFixtures(t *testing.T) {
	positive := []contactRequestFixtureOptions{
		{repositorySidecarConsumer: `type ReviewAlias = OptionsRepository
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositorySidecarConsumer: `type ReviewAlias = ReviewAliasTwo
type ReviewAliasTwo = ReviewAliasThree
type ReviewAliasThree = OptionsRepository
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositorySidecarConsumer: `type ReviewAlias = *OptionsRepository
func use(r ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositorySidecarConsumer: `type ReviewAlias = (*OptionsRepository)
func use(r ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{libConsumer: `import pg "database/postgres"
type ReviewAlias = pg.OptionsRepository
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositorySidecarConsumer: `type ReviewAlias = OptionsRepository
func use(r *ReviewAlias) { r.ReadContactRequestWorkflowSnapshot(nil) }`},
		{repositorySidecarConsumer: `type ReviewAlias = OptionsRepository
func use(r *ReviewAlias) { method := r.ReadContactRequestWorkflowSnapshot; _ = method }`},
	}
	for _, options := range positive {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, options) {
			t.Fatal("cannot create repository alias ownership fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 1 {
			t.Fatal("repository alias consumer was not owned by the canonical type")
		}
	}

	negative := []contactRequestFixtureOptions{
		{repositorySidecarConsumer: `type ReviewAlias OptionsRepository
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{libConsumer: `import fake "example.invalid/fake"
type ReviewAlias = fake.OptionsRepository
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositorySidecarConsumer: `type ReviewAlias = ReviewAliasTwo
type ReviewAliasTwo = ReviewAlias
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositorySidecarConsumer: `type ReviewAlias = MissingRepository
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{libConsumer: `func use(r struct{ ReadContactRequestWorkflowSnapshot int }) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{libConsumer: `func use(r interface{ ReadContactRequestWorkflowSnapshot() }) { _ = r.ReadContactRequestWorkflowSnapshot }`},
		{repositorySidecarConsumer: `type OtherRepository struct{ ReadContactRequestWorkflowSnapshot int }
type ReviewAlias = OtherRepository
func use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }`},
	}
	for _, options := range negative {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, options) {
			t.Fatal("cannot create repository alias rejection fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 0 {
			t.Fatal("repository alias scanner accepted a non-canonical type")
		}
	}
}

func TestContactRequestCrossFileOptionsRepositoryAliasOwnershipFixtures(t *testing.T) {
	positive := [][]contactRequestSource{
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = OptionsRepository\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = ReviewAliasTwo\n")},
			{relativePath: "database/postgres/review_alias_two.go", source: []byte("package postgres\ntype ReviewAliasTwo = OptionsRepository\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = (*OptionsRepository)\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "lib/review/review_alias.go", source: []byte("package review\nimport pg \"database/postgres\"\ntype ReviewAlias = pg.OptionsRepository\n")},
			{relativePath: "lib/review/review_consumer.go", source: []byte("package review\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = OptionsRepository\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r *ReviewAlias) { r.ReadContactRequestWorkflowSnapshot(nil) }\n")},
		},
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = OptionsRepository\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r *ReviewAlias) { method := r.ReadContactRequestWorkflowSnapshot; _ = method }\n")},
		},
	}
	for _, extraSources := range positive {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: extraSources}) {
			t.Fatal("cannot create cross-file repository alias fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 1 {
			t.Fatal("cross-file repository alias consumer was not resolved")
		}
	}

	negative := [][]contactRequestSource{
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias OptionsRepository\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = ReviewAliasTwo\n")},
			{relativePath: "database/postgres/review_alias_two.go", source: []byte("package postgres\ntype ReviewAliasTwo = ReviewAlias\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = MissingRepository\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "feature/one/review_alias.go", source: []byte("package review\ntype ReviewAlias = struct{}\n")},
			{relativePath: "feature/two/review_consumer.go", source: []byte("package review\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "lib/review/review_alias.go", source: []byte("package review\nimport fake \"example.invalid/fake\"\ntype ReviewAlias = fake.OptionsRepository\n")},
			{relativePath: "lib/review/review_consumer.go", source: []byte("package review\nfunc use(r *ReviewAlias) { _ = r.ReadContactRequestWorkflowSnapshot }\n")},
		},
		{
			{relativePath: "database/postgres/review_alias.go", source: []byte("package postgres\ntype ReviewAlias = OptionsRepository\n")},
			{relativePath: "database/postgres/review_consumer.go", source: []byte("package postgres\nfunc use() { ReviewAlias := struct{ ReadContactRequestWorkflowSnapshot int }{}; _ = ReviewAlias.ReadContactRequestWorkflowSnapshot }\n")},
		},
	}
	for _, extraSources := range negative {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: extraSources}) {
			t.Fatal("cannot create cross-file repository alias rejection fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 0 {
			t.Fatal("cross-file repository alias scanner accepted a non-canonical type")
		}
	}
}

func TestContactRequestNodeLevelOwnerFixtures(t *testing.T) {
	fixtures := []struct {
		options contactRequestFixtureOptions
		want    int
	}{
		{options: contactRequestFixtureOptions{}, want: 0},
		{options: contactRequestFixtureOptions{contractConsumer: `func reviewConsumer(value ContactRequestWorkflowSnapshot) { _ = value }`}, want: 1},
		{options: contactRequestFixtureOptions{repositoryConsumer: `func reviewConsumer(r *OptionsRepository) { _ = r.ReadContactRequestWorkflowSnapshot }`}, want: 1},
		{options: contactRequestFixtureOptions{repositoryConsumer: `func init() { var r *OptionsRepository; _ = r.ReadContactRequestWorkflowSnapshot }`}, want: 1},
		{options: contactRequestFixtureOptions{repositoryConsumer: `var reviewConsumer = (*OptionsRepository).ReadContactRequestWorkflowSnapshot`}, want: 1},
	}
	for _, fixture := range fixtures {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, fixture.options) {
			t.Fatal("cannot create node-level owner fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != fixture.want {
			t.Fatal("node-level owner allowance produced an unexpected consumer count")
		}
	}
}

func TestContactRequestSourceDiscoveryFixtures(t *testing.T) {
	root := t.TempDir()
	if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{}) {
		t.Fatal("cannot create complete discovery fixture")
	}
	if count, ok := scanContactRequestWorkspace(root); !ok || count != 0 {
		t.Fatal("complete minimal repository did not pass discovery")
	}

	productionConsumers := []contactRequestSource{
		{relativePath: "baserouter/review.go", source: contactRequestDiscoveryConsumerSource("baserouter")},
		{relativePath: "main/review.go", source: contactRequestDiscoveryConsumerSource("main")},
		{relativePath: "models/review/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "lib/review/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "database/review/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "controllers/review/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "feature/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "plugins/review/review.go", source: contactRequestDiscoveryConsumerSource("review")},
	}
	for _, consumer := range productionConsumers {
		root := t.TempDir()
		extraSources := []contactRequestSource{consumer}
		if consumer.relativePath == "plugins/review/review.go" {
			extraSources = append(extraSources, contactRequestSource{relativePath: "plugins/review/go.mod", source: []byte("module example.invalid/review\n")})
		}
		if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: extraSources}) {
			t.Fatal("cannot create full production discovery fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 1 {
			t.Fatal("full production discovery missed a local consumer")
		}
	}

	if _, ok := scanContactRequestWorkspace(t.TempDir()); ok {
		t.Fatal("empty repository passed discovery")
	}

	modulesOnly := t.TempDir()
	if !writeContactRequestFixtureFile(modulesOnly, "models/go.mod", "module models\n") ||
		!writeContactRequestFixtureFile(modulesOnly, "database/go.mod", "module database\n") {
		t.Fatal("cannot create module-only discovery fixture")
	}
	if _, ok := scanContactRequestWorkspace(modulesOnly); ok {
		t.Fatal("module-only repository passed discovery")
	}

	for _, missingRoot := range []string{"baserouter", "main", "models", "lib", "database", "controllers"} {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{omitRoot: missingRoot}) {
			t.Fatal("cannot create missing-root discovery fixture")
		}
		if _, ok := scanContactRequestWorkspace(root); ok {
			t.Fatal("repository with a missing critical root passed discovery")
		}
	}

	for _, options := range []contactRequestFixtureOptions{{emptyRoot: "baserouter"}, {testOnlyRoot: "baserouter"}, {emptyRoot: "lib"}, {testOnlyRoot: "lib"}} {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, options) {
			t.Fatal("cannot create empty-root discovery fixture")
		}
		if _, ok := scanContactRequestWorkspace(root); ok {
			t.Fatal("critical root without production declarations passed discovery")
		}
	}

	testOnly := t.TempDir()
	for _, directory := range []string{"baserouter", "main", "models", "lib", "database", "controllers"} {
		if !writeContactRequestFixtureFile(testOnly, pathpkg.Join(directory, "only_test.go"), "package fixture\nfunc TestOnly() {}\n") {
			t.Fatal("cannot create test-only discovery fixture")
		}
	}
	if !writeContactRequestFixtureFile(testOnly, "models/go.mod", "module models\n") ||
		!writeContactRequestFixtureFile(testOnly, "database/go.mod", "module database\n") {
		t.Fatal("cannot create test-only module fixture")
	}
	if _, ok := scanContactRequestWorkspace(testOnly); ok {
		t.Fatal("test-only repository passed discovery")
	}

	missingAnchors := []contactRequestFixtureOptions{
		{omitSnapshot: true},
		{omitReader: true},
		{omitOptionsRepository: true},
		{omitRepositoryMethod: true},
		{omitAddContactRequest: true},
	}
	for _, options := range missingAnchors {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, options) {
			t.Fatal("cannot create missing-anchor discovery fixture")
		}
		if _, ok := scanContactRequestWorkspace(root); ok {
			t.Fatal("repository with a missing canonical anchor passed discovery")
		}
	}
}

func TestContactRequestProductionDirectoryScopeFixtures(t *testing.T) {
	productionConsumers := []contactRequestSource{
		{relativePath: "lib/cache/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "feature/build/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "module/temp/review.go", source: contactRequestDiscoveryConsumerSource("review")},
		{relativePath: "feature/normal/review.go", source: contactRequestDiscoveryConsumerSource("review")},
	}
	for _, consumer := range productionConsumers {
		root := t.TempDir()
		if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: []contactRequestSource{consumer}}) {
			t.Fatal("cannot create production directory scope fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 1 {
			t.Fatal("production directory scope missed a normal local consumer")
		}
	}

	ignoredConsumers := []string{
		"feature/review_test.go",
		"vendor/review/review.go",
		".git/review/review.go",
	}
	for _, relativePath := range ignoredConsumers {
		root := t.TempDir()
		consumer := contactRequestSource{relativePath: relativePath, source: contactRequestDiscoveryConsumerSource("review")}
		if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: []contactRequestSource{consumer}}) {
			t.Fatal("cannot create excluded directory scope fixture")
		}
		count, ok := scanContactRequestWorkspace(root)
		if !ok || count != 0 {
			t.Fatal("production discovery scanned an excluded source")
		}
	}
}

func TestContactRequestProductionPathSafetyFixtures(t *testing.T) {
	root := t.TempDir()
	distinct := []contactRequestSource{
		{relativePath: "feature/one.go", source: contactRequestDiscoveryConsumerSource("feature")},
		{relativePath: "feature/two.go", source: contactRequestDiscoveryConsumerSource("feature")},
	}
	if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: distinct}) {
		t.Fatal("cannot create distinct production path fixture")
	}
	if count, ok := scanContactRequestWorkspace(root); !ok || count != 2 {
		t.Fatal("distinct normal production files were not scanned independently")
	}

	root = t.TempDir()
	consumer := contactRequestSource{relativePath: "feature/review.go", source: contactRequestDiscoveryConsumerSource("feature")}
	if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: []contactRequestSource{consumer}}) {
		t.Fatal("cannot create duplicate production path fixture")
	}
	if count, ok := scanContactRequestWorkspaceFromRoots(root, []string{root, filepath.Clean(root)}); !ok || count != 1 {
		t.Fatal("duplicate canonical production path was counted more than once")
	}

	externalRoot := t.TempDir()
	if !writeContactRequestFixtureFile(externalRoot, "outside.go", string(contactRequestDiscoveryConsumerSource("outside"))) {
		t.Fatal("cannot create external symlink target fixture")
	}
	root = t.TempDir()
	if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{}) {
		t.Fatal("cannot create external symlink repository fixture")
	}
	if !contactRequestSymlinkDiscoveryFails(root, filepath.Join(externalRoot, "outside.go"), "feature/external.go") {
		t.Fatal("external production symlink did not fail closed")
	}

	root = t.TempDir()
	internal := contactRequestSource{relativePath: "feature/target.go", source: contactRequestDiscoveryConsumerSource("feature")}
	if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{extraSources: []contactRequestSource{internal}}) {
		t.Fatal("cannot create internal symlink repository fixture")
	}
	if !contactRequestSymlinkDiscoveryFails(root, filepath.Join(root, "feature", "target.go"), "feature/internal.go") {
		t.Fatal("internal production symlink did not fail closed")
	}

	externalDirectory := t.TempDir()
	if !writeContactRequestFixtureFile(externalDirectory, "review.go", string(contactRequestDiscoveryConsumerSource("review"))) {
		t.Fatal("cannot create directory symlink target fixture")
	}
	root = t.TempDir()
	if !writeContactRequestFixtureRepository(root, contactRequestFixtureOptions{}) {
		t.Fatal("cannot create directory symlink repository fixture")
	}
	if !contactRequestSymlinkDiscoveryFails(root, externalDirectory, "feature/linked") {
		t.Fatal("production directory symlink did not fail closed")
	}
}

func contactRequestSymlinkDiscoveryFails(root, target, relativeLink string) bool {
	linkPath := filepath.Join(root, filepath.FromSlash(relativeLink))
	if err := os.MkdirAll(filepath.Dir(linkPath), 0700); err != nil {
		return false
	}
	if err := os.Symlink(target, linkPath); err != nil {
		return false
	}
	_, ok := scanContactRequestWorkspace(root)
	return !ok
}

func contactRequestDiscoveryConsumerSource(packageName string) []byte {
	return []byte("package " + packageName + "\nimport \"models/data\"\nvar reviewConsumer data.ContactRequestWorkflowSnapshot\n")
}

type contactRequestFixtureOptions struct {
	omitRoot                  string
	emptyRoot                 string
	testOnlyRoot              string
	omitSnapshot              bool
	omitReader                bool
	omitOptionsRepository     bool
	omitRepositoryMethod      bool
	omitAddContactRequest     bool
	contractConsumer          string
	repositoryConsumer        string
	repositorySidecarConsumer string
	libConsumer               string
	extraSources              []contactRequestSource
}

func writeContactRequestFixtureRepository(root string, options contactRequestFixtureOptions) bool {
	for _, directory := range []string{"baserouter", "main", "models", "lib", "database", "controllers"} {
		if directory == options.omitRoot {
			continue
		}
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			return false
		}
		if directory == options.testOnlyRoot && !writeContactRequestFixtureFile(root, pathpkg.Join(directory, "only_test.go"), "package "+directory+"\nfunc TestOnly() {}\n") {
			return false
		}
	}

	if options.omitRoot != "models" && !writeContactRequestFixtureFile(root, "models/go.mod", "module models\n") {
		return false
	}
	if options.omitRoot != "database" && !writeContactRequestFixtureFile(root, "database/go.mod", "module database\n") {
		return false
	}
	if options.omitRoot != "baserouter" && options.emptyRoot != "baserouter" && options.testOnlyRoot != "baserouter" &&
		!writeContactRequestFixtureFile(root, "baserouter/baserouter.go", "package baserouter\ntype Marker struct{}\n") {
		return false
	}
	if options.omitRoot != "main" && options.emptyRoot != "main" && options.testOnlyRoot != "main" &&
		!writeContactRequestFixtureFile(root, "main/main.go", "package main\nfunc main() {}\n") {
		return false
	}
	if options.omitRoot != "models" && options.emptyRoot != "models" && options.testOnlyRoot != "models" {
		if !writeContactRequestFixtureFile(root, "models/models.go", "package models\ntype Marker struct{}\n") {
			return false
		}
		contract := "package data\nimport \"context\"\n"
		if !options.omitSnapshot {
			contract += "type ContactRequestWorkflowSnapshot struct{}\n"
		}
		if !options.omitReader {
			contract += "type ContactRequestWorkflowSnapshotReader interface { ReadContactRequestWorkflowSnapshot(context.Context) (ContactRequestWorkflowSnapshot, bool, error) }\n"
		}
		contract += options.contractConsumer + "\n"
		if !writeContactRequestFixtureFile(root, "models/data/contact_request_workflow_snapshot.go", contract) {
			return false
		}
	}
	if options.omitRoot != "lib" && options.emptyRoot != "lib" && options.testOnlyRoot != "lib" {
		libSource := "package lib\n" + options.libConsumer + "\ntype Marker struct{}\n"
		if !writeContactRequestFixtureFile(root, "lib/lib.go", libSource) {
			return false
		}
	}
	if options.omitRoot != "database" && options.emptyRoot != "database" && options.testOnlyRoot != "database" {
		if !writeContactRequestFixtureFile(root, "database/database.go", "package database\ntype Marker struct{}\n") {
			return false
		}
		optionsSource := "package postgres\n"
		if !options.omitOptionsRepository {
			optionsSource += "type OptionsRepository struct{}\n"
		}
		if !writeContactRequestFixtureFile(root, "database/postgres/options.go", optionsSource) {
			return false
		}
		repository := "package postgres\nimport (\"context\"; \"models/data\")\nvar _ data.ContactRequestWorkflowSnapshotReader = (*OptionsRepository)(nil)\n"
		if !options.omitRepositoryMethod {
			repository += "func (r *OptionsRepository) ReadContactRequestWorkflowSnapshot(context.Context) (data.ContactRequestWorkflowSnapshot, bool, error) { return data.ContactRequestWorkflowSnapshot{}, false, nil }\n"
		}
		repository += options.repositoryConsumer + "\n"
		if !writeContactRequestFixtureFile(root, "database/postgres/contact_request_workflow_snapshot.go", repository) {
			return false
		}
		if options.repositorySidecarConsumer != "" && !writeContactRequestFixtureFile(root, "database/postgres/review.go", "package postgres\n"+options.repositorySidecarConsumer+"\n") {
			return false
		}
	}
	if options.omitRoot != "controllers" && options.emptyRoot != "controllers" && options.testOnlyRoot != "controllers" {
		if !writeContactRequestFixtureFile(root, "controllers/controllers.go", "package controllers\ntype Marker struct{}\n") {
			return false
		}
		postSource := "package post\n"
		if !options.omitAddContactRequest {
			postSource += "func AddContactRequest() {}\n"
		}
		if !writeContactRequestFixtureFile(root, "controllers/post/post.go", postSource) {
			return false
		}
	}
	for _, source := range options.extraSources {
		if !writeContactRequestFixtureFile(root, source.relativePath, string(source.source)) {
			return false
		}
	}
	return true
}

func writeContactRequestFixtureFile(root, relativePath, contents string) bool {
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return false
	}
	return os.WriteFile(path, []byte(contents), 0600) == nil
}

func TestAddContactRequestUsesOnlyApprovedOwnedReader(t *testing.T) {
	root := contactRequestWorkspaceRoot(t)
	path := filepath.Join(root, "controllers", "post", "post.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read contact-request caller")
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatal("cannot parse contact-request caller")
	}
	if !hasExactAddContactRequestSnapshotAssignmentCall(file) {
		t.Fatal("AddContactRequest owned snapshot call shape changed")
	}
}

func TestAddContactRequestOwnedCallGuardFixtures(t *testing.T) {
	positive := `package fixture
func AddContactRequest() func() {
	return func() {
		contactRequestSnapshot, err := contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader)
		_, _ = contactRequestSnapshot, err
	}
}`
	negative := []string{
		`package fixture
func AddContactRequest() func() { return func() { _ = contactrequestsnapshot.Read } }`,
		`package fixture
func AddContactRequest() func() { return func() { fn := contactrequestsnapshot.Read; _ = fn } }`,
		`package fixture
func AddContactRequest() func() { return func() {} }
func other() { contactRequestSnapshot, err := contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader); _, _ = contactRequestSnapshot, err }`,
		`package fixture
func AddContactRequest() func() {
	return func() {
		contactRequestSnapshot, err := contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader)
		contactRequestSnapshot, err = contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader)
		_, _ = contactRequestSnapshot, err
	}
}`,
		`package fixture
func AddContactRequest() func() {
	return func() {
		contactRequestSnapshot, err := other.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader)
		_, _ = contactRequestSnapshot, err
	}
}`,
		`package fixture
func AddContactRequest() func() {
	return func() {
		contactRequestSnapshot, err := contactrequestsnapshot.Read(context.Background(), utilities.ContactRequestWorkflowSnapshotReader)
		_, _ = contactRequestSnapshot, err
	}
}`,
		`package fixture
func AddContactRequest() func() {
	return func() {
		contactRequestSnapshot, err := contactrequestsnapshot.Read(c.UserContext(), utilities.OtherReader)
		_, _ = contactRequestSnapshot, err
	}
}`,
	}
	file, ok := parseContactRequestFixture(positive)
	if !ok || !hasExactAddContactRequestSnapshotAssignmentCall(file) {
		t.Fatal("owned call guard rejected the exact assignment call")
	}
	for _, source := range negative {
		file, ok := parseContactRequestFixture(source)
		if !ok || hasExactAddContactRequestSnapshotAssignmentCall(file) {
			t.Fatal("owned call guard accepted an invalid call shape")
		}
	}
}

func TestContactRequestHelperReferenceScannerFixtures(t *testing.T) {
	config := contactRequestScannerConfig{
		helperImportPath: "post/contactrequestsnapshot",
		addContactPath:   "controllers/post/post.go",
		helperPath:       "controllers/post/contactrequestsnapshot/decision.go",
	}
	approved := []contactRequestSource{
		{relativePath: config.helperPath, source: []byte(`package contactrequestsnapshot
func Read(any, any) (any, error) { return nil, nil }`)},
		{relativePath: config.addContactPath, source: []byte(`package post
import "post/contactrequestsnapshot"
func AddContactRequest() {
	contactRequestSnapshot, err := contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader)
	_, _ = contactRequestSnapshot, err
}`)},
	}
	if !contactRequestHelperReferencesAreApproved(approved, config) {
		t.Fatal("approved helper reference fixture was rejected")
	}

	mutations := [][]contactRequestSource{
		{{relativePath: "controllers/post/leaked.go", source: []byte(`package post
import "post/contactrequestsnapshot"
func leakedContactReader() any { return contactrequestsnapshot.Read }`)}},
		{{relativePath: "controllers/post/second.go", source: []byte(`package post
import "post/contactrequestsnapshot"
func AddContactRequestSecond() { _, _ = contactrequestsnapshot.Read(c.UserContext(), utilities.ContactRequestWorkflowSnapshotReader) }`)}},
		{{relativePath: "controllers/post/alias.go", source: []byte(`package post
import "post/contactrequestsnapshot"
func leaked() { fn := contactrequestsnapshot.Read; _ = fn }`)}},
		{{relativePath: "controllers/post/package_value.go", source: []byte(`package post
import "post/contactrequestsnapshot"
var leaked = contactrequestsnapshot.Read`)}},
		{{relativePath: "controllers/post/struct.go", source: []byte(`package post
import "post/contactrequestsnapshot"
var leaked = struct{ Read any }{Read: contactrequestsnapshot.Read}`)}},
		{{relativePath: "controllers/post/callback.go", source: []byte(`package post
import "post/contactrequestsnapshot"
func consume(any) {}
func leaked() { consume(contactrequestsnapshot.Read) }`)}},
		{{relativePath: "controllers/post/explicit.go", source: []byte(`package post
import helper "post/contactrequestsnapshot"
func leaked() any { return helper.Read }`)}},
		{{relativePath: "controllers/post/dot.go", source: []byte(`package post
import . "post/contactrequestsnapshot"
func leaked() any { return Read }`)}},
		{
			{relativePath: "controllers/post/alias_source.go", source: []byte(`package post
import "post/contactrequestsnapshot"
var helperAlias = contactrequestsnapshot.Read`)},
			{relativePath: "controllers/post/alias_consumer.go", source: []byte(`package post
var leaked = helperAlias`)},
		},
		{{relativePath: "controllers/post/contactrequestsnapshot/leaked.go", source: []byte(`package contactrequestsnapshot
func leaked() any { return Read }`)}},
		{{relativePath: "controllers/post/parenthesized.go", source: []byte(`package post
import "post/contactrequestsnapshot"
func leaked() any { return (contactrequestsnapshot.Read) }`)}},
	}
	for _, mutation := range mutations {
		sources := append(append([]contactRequestSource{}, approved...), mutation...)
		if contactRequestHelperReferencesAreApproved(sources, config) {
			t.Fatal("forbidden helper reference fixture was accepted")
		}
	}

	safe := append(append([]contactRequestSource{}, approved...),
		contactRequestSource{relativePath: "controllers/post/shadow.go", source: []byte(`package post
import "post/contactrequestsnapshot"
func shadow(contactrequestsnapshot struct{ Read any }) any { return contactrequestsnapshot.Read }`)},
		contactRequestSource{relativePath: "controllers/post/contactrequestsnapshot/rand.go", source: []byte(`package contactrequestsnapshot
import "crypto/rand"
func unrelatedRead(buffer []byte) (int, error) { return rand.Read(buffer) }`)},
		contactRequestSource{relativePath: "controllers/post/contactrequestsnapshot/io.go", source: []byte(`package contactrequestsnapshot
import "io"
func readerRead(reader io.Reader, buffer []byte) (int, error) { return reader.Read(buffer) }`)},
		contactRequestSource{relativePath: "controllers/post/contactrequestsnapshot/local.go", source: []byte(`package contactrequestsnapshot
type localReadField struct{ Read any }
type localReadMethod struct{}
func (localReadMethod) Read() {}
func useLocalReadField(value localReadField) any { return value.Read }
func useLocalReadMethod(value localReadMethod) { value.Read() }
func useLocalReadVariable() { Read := func() {}; Read() }
func useLocalReadParameter(Read func()) { Read() }`)},
		contactRequestSource{relativePath: "controllers/post/other.go", source: []byte(`package post
import other "example.invalid/other"
var otherRead = other.Read
const text = "contactrequestsnapshot.Read"
// contactrequestsnapshot.Read
`)},
		contactRequestSource{relativePath: "controllers/post/ignored_test.go", source: []byte(`package post
import "post/contactrequestsnapshot"
var ignored = contactrequestsnapshot.Read`)},
	)
	if !contactRequestHelperReferencesAreApproved(safe, config) {
		t.Fatal("safe helper reference fixture was rejected")
	}
}

func TestContactRequestWiringDiagnosticsAreConstant(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contact-request snapshot test")
	}
	file, err := parser.ParseFile(token.NewFileSet(), currentFile, nil, 0)
	if err != nil {
		t.Fatal("cannot parse contact-request snapshot test")
	}
	valid := true
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch selector.Sel.Name {
		case "Fatal" + "f", "Error" + "f", "Log" + "f":
			valid = false
		case "Fatal", "Error", "Log":
			if len(call.Args) != 1 {
				valid = false
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				valid = false
			}
		}
		return true
	})
	if !valid {
		t.Fatal("wiring test diagnostics must remain constant")
	}
}

func hasExactAddContactRequestSnapshotAssignmentCall(file *ast.File) bool {
	functionCount := 0
	callCount := 0
	candidateCallCount := 0
	assignmentCount := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || function.Name.Name != "AddContactRequest" {
			continue
		}
		functionCount++
		handlerBody, ok := contactRequestHandlerBody(function)
		if !ok {
			continue
		}
		ast.Inspect(handlerBody, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if ok {
				selector, selectorOK := call.Fun.(*ast.SelectorExpr)
				if selectorOK {
					receiver, receiverOK := selector.X.(*ast.Ident)
					if receiverOK && receiver.Name == "contactrequestsnapshot" && selector.Sel.Name == "Read" {
						candidateCallCount++
					}
				}
			}
			if ok && isExactContactRequestSnapshotCall(call) {
				callCount++
			}
			assignment, ok := node.(*ast.AssignStmt)
			if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
				return true
			}
			options, optionsOK := assignment.Lhs[0].(*ast.Ident)
			errValue, errOK := assignment.Lhs[1].(*ast.Ident)
			call, callOK := assignment.Rhs[0].(*ast.CallExpr)
			if optionsOK && errOK && callOK && options.Name == "contactRequestSnapshot" && errValue.Name == "err" && isExactContactRequestSnapshotCall(call) {
				assignmentCount++
			}
			return true
		})
	}
	return functionCount == 1 && candidateCallCount == 1 && callCount == 1 && assignmentCount == 1
}

func contactRequestHandlerBody(function *ast.FuncDecl) (*ast.BlockStmt, bool) {
	if len(function.Body.List) != 1 {
		return nil, false
	}
	result, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return nil, false
	}
	handler, ok := result.Results[0].(*ast.FuncLit)
	if !ok {
		return nil, false
	}
	return handler.Body, true
}

func isExactContactRequestSnapshotCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Read" || len(call.Args) != 2 {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok || receiver.Name != "contactrequestsnapshot" || receiver.Obj != nil {
		return false
	}
	contextCall, ok := call.Args[0].(*ast.CallExpr)
	if !ok || len(contextCall.Args) != 0 {
		return false
	}
	contextSelector, ok := contextCall.Fun.(*ast.SelectorExpr)
	if !ok || contextSelector.Sel.Name != "UserContext" {
		return false
	}
	contextReceiver, ok := contextSelector.X.(*ast.Ident)
	if !ok || contextReceiver.Name != "c" {
		return false
	}
	reader, ok := call.Args[1].(*ast.SelectorExpr)
	if !ok || reader.Sel.Name != contactRequestSnapshotReader {
		return false
	}
	utilities, ok := reader.X.(*ast.Ident)
	return ok && utilities.Name == "utilities"
}

type contactRequestHelperImportBindings struct {
	aliases map[string]bool
	dot     bool
}

func hasOnlyApprovedContactRequestHelperReferences(root string) bool {
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		return false
	}
	config, ok := contactRequestScannerConfigForWorkspace(root)
	if !ok {
		return false
	}
	config.helperImportPath, ok = canonicalContactRequestImportPath(root, "controllers/post", "contactrequestsnapshot")
	if !ok {
		return false
	}
	config.helperPath = "controllers/post/contactrequestsnapshot/decision.go"
	return contactRequestHelperReferencesAreApproved(sources, config)
}

func contactRequestHelperReferencesAreApproved(sources []contactRequestSource, config contactRequestScannerConfig) bool {
	parsed := []contactRequestParsedSource{}
	var helperDeclaration *ast.FuncDecl
	for _, source := range sources {
		if strings.HasSuffix(source.relativePath, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), source.relativePath, source.source, 0)
		if err != nil {
			return false
		}
		parsedSource := contactRequestParsedSource{relativePath: source.relativePath, file: file}
		parsed = append(parsed, parsedSource)
		if source.relativePath != config.helperPath || file.Name.Name != "contactrequestsnapshot" {
			continue
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Name.Name != "Read" {
				continue
			}
			if helperDeclaration != nil {
				return false
			}
			helperDeclaration = function
		}
	}
	if helperDeclaration == nil {
		return false
	}

	references := 0
	approved := 0
	valid := true
	for _, source := range parsed {
		bindings := contactRequestHelperImports(source.file, config.helperImportPath)
		declarations := contactRequestDeclarationIdentifiers(source.file)
		selectorIdentifiers := map[*ast.Ident]bool{}
		ast.Inspect(source.file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok {
				selectorIdentifiers[selector.Sel] = true
			}
			return true
		})
		stack := []ast.Node{}
		ast.Inspect(source.file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			parent := ast.Node(nil)
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			isReference := false
			switch typed := node.(type) {
			case *ast.SelectorExpr:
				receiver, ok := typed.X.(*ast.Ident)
				isReference = ok && typed.Sel.Name == "Read" && bindings.aliases[receiver.Name] && (receiver.Obj == nil || receiver.Obj.Kind == ast.Pkg)
			case *ast.Ident:
				if typed != helperDeclaration.Name && !selectorIdentifiers[typed] && !declarations[typed] && typed.Name == "Read" {
					samePackage := contactRequestSourceIsPackage(source, pathpkg.Dir(config.helperPath), "contactrequestsnapshot")
					isReference = (bindings.dot && typed.Obj == nil) || (samePackage && (typed.Obj == nil || typed.Obj.Decl == helperDeclaration))
				}
			}
			if isReference {
				references++
				call, callOK := parent.(*ast.CallExpr)
				function := enclosingContactRequestFunction(stack)
				if source.relativePath == config.addContactPath && callOK && call.Fun == node && function != nil && function.Name.Name == "AddContactRequest" && isExactContactRequestSnapshotCall(call) {
					approved++
				} else {
					valid = false
				}
			}
			stack = append(stack, node)
			return true
		})
	}
	return valid && references == 1 && approved == 1
}

func contactRequestHelperImports(file *ast.File, helperImportPath string) contactRequestHelperImportBindings {
	bindings := contactRequestHelperImportBindings{aliases: map[string]bool{}}
	for _, imported := range file.Imports {
		importPath, err := strconv.Unquote(imported.Path.Value)
		if err != nil || importPath != helperImportPath {
			continue
		}
		alias := pathpkg.Base(importPath)
		if imported.Name != nil {
			alias = imported.Name.Name
		}
		if alias == "." {
			bindings.dot = true
		} else if alias != "_" {
			bindings.aliases[alias] = true
		}
	}
	return bindings
}

func contactRequestDeclarationIdentifiers(file *ast.File) map[*ast.Ident]bool {
	declarations := map[*ast.Ident]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.ImportSpec:
			if typed.Name != nil {
				declarations[typed.Name] = true
			}
		case *ast.TypeSpec:
			declarations[typed.Name] = true
		case *ast.FuncDecl:
			declarations[typed.Name] = true
		case *ast.Field:
			for _, name := range typed.Names {
				declarations[name] = true
			}
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				declarations[name] = true
			}
		case *ast.AssignStmt:
			if typed.Tok == token.DEFINE {
				for _, expression := range typed.Lhs {
					if name, ok := expression.(*ast.Ident); ok {
						declarations[name] = true
					}
				}
			}
		case *ast.RangeStmt:
			if typed.Tok == token.DEFINE {
				if name, ok := typed.Key.(*ast.Ident); ok {
					declarations[name] = true
				}
				if name, ok := typed.Value.(*ast.Ident); ok {
					declarations[name] = true
				}
			}
		}
		return true
	})
	return declarations
}

func enclosingContactRequestFunction(stack []ast.Node) *ast.FuncDecl {
	for index := len(stack) - 1; index >= 0; index-- {
		if function, ok := stack[index].(*ast.FuncDecl); ok {
			return function
		}
	}
	return nil
}

func hasExactApprovedContactRequestWiring(root string) bool {
	modelsFile, ok := parseContactRequestProductionFile(root, "models/models.go")
	if !ok || modelsFile.Name.Name != "models" || !hasExactUtilitiesContactRequestReader(modelsFile) {
		return false
	}
	mainFile, ok := parseContactRequestProductionFile(root, "main/main.go")
	if !ok || mainFile.Name.Name != "main" || !hasExactMainContactRequestInjection(mainFile) {
		return false
	}
	helperFile, ok := parseContactRequestProductionFile(root, "controllers/post/contactrequestsnapshot/decision.go")
	if !ok || helperFile.Name.Name != "contactrequestsnapshot" || !hasExactContactRequestHelper(helperFile) {
		return false
	}
	postFile, ok := parseContactRequestProductionFile(root, "controllers/post/post.go")
	return ok && postFile.Name.Name == "post" && hasExactImport(postFile, "post/contactrequestsnapshot") && hasExactAddContactRequestSnapshotAssignmentCall(postFile)
}

func hasExactApprovedContactRequestSymbolInventory(root string) bool {
	sources, ok := contactRequestProductionSources(root)
	if !ok {
		return false
	}
	readerSelectors := 0
	helperImports := 0
	helperCalls := 0
	readerMethodCalls := 0
	constructorCalls := 0
	for _, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), source.relativePath, source.source, 0)
		if err != nil {
			return false
		}
		for _, imported := range file.Imports {
			value, err := strconv.Unquote(imported.Path.Value)
			if err == nil && value == "post/contactrequestsnapshot" {
				helperImports++
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.SelectorExpr:
				if typed.Sel.Name == contactRequestSnapshotReader {
					readerSelectors++
				}
				if typed.Sel.Name == contactRequestSnapshotMethod {
					readerMethodCalls++
				}
			case *ast.CallExpr:
				selector, selectorOK := typed.Fun.(*ast.SelectorExpr)
				if selectorOK {
					receiver, receiverOK := selector.X.(*ast.Ident)
					if receiverOK && receiver.Name == "contactrequestsnapshot" && selector.Sel.Name == "Read" {
						helperCalls++
					}
					if selector.Sel.Name == "NewOptionsRepository" {
						constructorCalls++
					}
				}
			}
			return true
		})
	}
	return readerSelectors == 6 && helperImports == 1 && helperCalls == 1 && readerMethodCalls == 1 && constructorCalls == 1
}

func parseContactRequestProductionFile(root, relativePath string) (*ast.File, bool) {
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	return file, err == nil
}

func hasExactUtilitiesContactRequestReader(file *ast.File) bool {
	if !hasExactImport(file, "models/data") {
		return false
	}
	count := 0
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Utilities" {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return false
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 1 && field.Names[0].Name == contactRequestSnapshotReader && contactRequestNodeText(field.Type) == "data."+contactRequestSnapshotReader {
					count++
				}
			}
		}
	}
	return count == 1
}

func hasExactMainContactRequestInjection(file *ast.File) bool {
	run := findContactRequestFunction(file, "run")
	if run == nil {
		return false
	}
	repositoryAssignments := 0
	injections := 0
	var repositoryObject *ast.Object
	ast.Inspect(run, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return true
		}
		if assignment.Tok == token.DEFINE && contactRequestNodeText(assignment.Rhs[0]) == "postgres.NewOptionsRepository(pool)" {
			name, nameOK := assignment.Lhs[0].(*ast.Ident)
			if nameOK && name.Name == "optionsRepository" {
				repositoryAssignments++
				repositoryObject = name.Obj
			}
		}
		left, leftOK := assignment.Lhs[0].(*ast.SelectorExpr)
		right, rightOK := assignment.Rhs[0].(*ast.Ident)
		if assignment.Tok == token.ASSIGN && leftOK && rightOK && contactRequestNodeText(left) == "utilities."+contactRequestSnapshotReader && right.Name == "optionsRepository" {
			if repositoryObject != nil && right.Obj == repositoryObject {
				injections++
			}
		}
		return true
	})
	return repositoryAssignments == 1 && injections == 1
}

func hasExactContactRequestHelper(file *ast.File) bool {
	if !hasExactImport(file, "models/data") {
		return false
	}
	readCount := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "Read" {
			continue
		}
		readCount++
		if function.Recv != nil || function.Type.Params == nil || len(function.Type.Params.List) != 2 || function.Type.Results == nil || len(function.Type.Results.List) != 2 {
			return false
		}
		if contactRequestNodeText(function.Type.Params.List[0].Type) != "context.Context" || contactRequestNodeText(function.Type.Params.List[1].Type) != "data."+contactRequestSnapshotReader {
			return false
		}
		if contactRequestNodeText(function.Type.Results.List[0].Type) != "data."+contactRequestSnapshotType || contactRequestNodeText(function.Type.Results.List[1].Type) != "error" {
			return false
		}
	}
	return readCount == 1
}

func hasExactImport(file *ast.File, importPath string) bool {
	count := 0
	for _, imported := range file.Imports {
		value, err := strconv.Unquote(imported.Path.Value)
		if err == nil && value == importPath && (imported.Name == nil || imported.Name.Name == pathpkg.Base(importPath)) {
			count++
		}
	}
	return count == 1
}

func findContactRequestFunction(file *ast.File, name string) *ast.FuncDecl {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == name {
			return function
		}
	}
	return nil
}

func contactRequestNodeText(node ast.Node) string {
	var buffer bytes.Buffer
	if format.Node(&buffer, token.NewFileSet(), node) != nil {
		return ""
	}
	return buffer.String()
}

func contactRequestScannerConfigForWorkspace(root string) (contactRequestScannerConfig, bool) {
	dataImportPath, ok := canonicalContactRequestImportPath(root, "models", "data")
	if !ok {
		return contactRequestScannerConfig{}, false
	}
	repositoryImportPath, ok := canonicalContactRequestImportPath(root, "database", "postgres")
	if !ok {
		return contactRequestScannerConfig{}, false
	}
	return contactRequestScannerConfig{
		dataImportPath:       dataImportPath,
		repositoryImportPath: repositoryImportPath,
		contractPath:         "models/data/contact_request_workflow_snapshot.go",
		repositoryPath:       "database/postgres/contact_request_workflow_snapshot.go",
		optionsPath:          "database/postgres/options.go",
		addContactPath:       "controllers/post/post.go",
	}, true
}

func canonicalContactRequestImportPath(root, moduleDirectory, packageDirectory string) (string, bool) {
	moduleRoot := filepath.Join(root, moduleDirectory)
	moduleFile, err := os.ReadFile(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		return "", false
	}
	modulePath := ""
	for _, line := range strings.Split(string(moduleFile), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			modulePath = fields[1]
			break
		}
	}
	if modulePath == "" {
		return "", false
	}
	packagePath := filepath.Join(moduleRoot, packageDirectory)
	info, err := os.Stat(packagePath)
	if err != nil || !info.IsDir() {
		return "", false
	}
	relative, err := filepath.Rel(moduleRoot, packagePath)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
		return "", false
	}
	return pathpkg.Join(modulePath, filepath.ToSlash(relative)), true
}

func scanContactRequestWorkspace(root string) (int, bool) {
	sources, ok := contactRequestProductionSources(root)
	return scanContactRequestWorkspaceSources(root, sources, ok)
}

func scanContactRequestWorkspaceFromRoots(root string, scanRoots []string) (int, bool) {
	sources, ok := contactRequestProductionSourcesFromRoots(root, scanRoots)
	return scanContactRequestWorkspaceSources(root, sources, ok)
}

func scanContactRequestWorkspaceSources(root string, sources []contactRequestSource, ok bool) (int, bool) {
	if !ok {
		return 0, false
	}
	config, ok := contactRequestScannerConfigForWorkspace(root)
	if !ok {
		return 0, false
	}
	context, ok := prepareContactRequestScanContext(sources, config)
	if !ok || !contactRequestAnchorsAreComplete(context) {
		return 0, false
	}
	return countContactRequestReferences(context), true
}

func contactRequestProductionSources(root string) ([]contactRequestSource, bool) {
	return contactRequestProductionSourcesFromRoots(root, []string{root})
}

func contactRequestProductionSourcesFromRoots(root string, scanRoots []string) ([]contactRequestSource, bool) {
	requiredRoots := []string{"baserouter", "main", "models", "lib", "database", "controllers"}
	requiredProduction := map[string]bool{}
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return nil, false
	}
	rootAbsolute = filepath.Clean(rootAbsolute)
	rootInfo, err := os.Lstat(rootAbsolute)
	if err != nil || !rootInfo.IsDir() || contactRequestUnsafeProductionMode(rootInfo.Mode()) {
		return nil, false
	}
	for _, requiredRoot := range requiredRoots {
		rootPath := filepath.Join(rootAbsolute, requiredRoot)
		info, err := os.Lstat(rootPath)
		if err != nil || !info.IsDir() || contactRequestUnsafeProductionMode(info.Mode()) {
			return nil, false
		}
		requiredProduction[requiredRoot] = false
	}

	sources := []contactRequestSource{}
	seenPaths := map[string]bool{}
	for _, scanRoot := range scanRoots {
		scanRootAbsolute, ok := contactRequestPathWithinRoot(rootAbsolute, scanRoot)
		if !ok {
			return nil, false
		}
		scanRootInfo, err := os.Lstat(scanRootAbsolute)
		if err != nil || !scanRootInfo.IsDir() || contactRequestUnsafeProductionMode(scanRootInfo.Mode()) {
			return nil, false
		}
		err = filepath.WalkDir(scanRootAbsolute, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			absolutePath, ok := contactRequestPathWithinRoot(rootAbsolute, filePath)
			if !ok {
				return fs.ErrInvalid
			}
			info, err := os.Lstat(absolutePath)
			if err != nil || contactRequestUnsafeProductionMode(info.Mode()) {
				return fs.ErrInvalid
			}
			if entry.IsDir() {
				if absolutePath != rootAbsolute && contactRequestExcludedProductionDirectory(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			if !info.Mode().IsRegular() {
				return fs.ErrInvalid
			}
			canonicalPath := absolutePath
			if runtime.GOOS == "windows" {
				canonicalPath = strings.ToLower(canonicalPath)
			}
			if seenPaths[canonicalPath] {
				return nil
			}
			seenPaths[canonicalPath] = true
			relative, err := filepath.Rel(rootAbsolute, absolutePath)
			if err != nil || relative == "." {
				return fs.ErrInvalid
			}
			relative = filepath.ToSlash(relative)
			source, err := os.ReadFile(absolutePath)
			if err != nil {
				return err
			}
			if contactRequestSourceHasProductionDeclaration(relative, source) {
				rootName := strings.SplitN(relative, "/", 2)[0]
				if _, required := requiredProduction[rootName]; required {
					requiredProduction[rootName] = true
				}
			}
			sources = append(sources, contactRequestSource{relativePath: relative, source: source})
			return nil
		})
		if err != nil {
			return nil, false
		}
	}
	if len(sources) == 0 {
		return nil, false
	}
	for _, requiredRoot := range requiredRoots {
		if !requiredProduction[requiredRoot] {
			return nil, false
		}
	}
	return sources, true
}

func contactRequestPathWithinRoot(rootAbsolute, candidate string) (string, bool) {
	candidateAbsolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	candidateAbsolute = filepath.Clean(candidateAbsolute)
	relative, err := filepath.Rel(rootAbsolute, candidateAbsolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return candidateAbsolute, true
}

func contactRequestUnsafeProductionMode(mode fs.FileMode) bool {
	return mode&os.ModeSymlink != 0
}

func contactRequestExcludedProductionDirectory(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "vendor":
		return true
	}
	return false
}

func contactRequestSourceHasProductionDeclaration(path string, source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		return false
	}
	for _, declaration := range file.Decls {
		switch typed := declaration.(type) {
		case *ast.FuncDecl:
			return true
		case *ast.GenDecl:
			if typed.Tok != token.IMPORT && len(typed.Specs) != 0 {
				return true
			}
		}
	}
	return false
}

func prepareContactRequestScanContext(sources []contactRequestSource, config contactRequestScannerConfig) (contactRequestScanContext, bool) {
	context := contactRequestScanContext{
		config:       config,
		packageTypes: map[contactRequestPackageIdentity]map[string][]contactRequestPackageType{},
		allowedNodes: map[ast.Node]bool{},
	}
	for _, source := range sources {
		if strings.HasSuffix(source.relativePath, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), source.relativePath, source.source, 0)
		if err != nil {
			return contactRequestScanContext{}, false
		}
		context.sources = append(context.sources, contactRequestParsedSource{relativePath: source.relativePath, file: file})
	}
	if len(context.sources) == 0 {
		return contactRequestScanContext{}, false
	}
	for _, source := range context.sources {
		identity := contactRequestPackageIdentityForSource(source)
		if context.packageTypes[identity] == nil {
			context.packageTypes[identity] = map[string][]contactRequestPackageType{}
		}
		for _, declaration := range source.file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok {
					continue
				}
				entry := contactRequestPackageType{source: source, declaration: typeSpec}
				context.packageTypes[identity][typeSpec.Name.Name] = append(context.packageTypes[identity][typeSpec.Name.Name], entry)
			}
		}
	}

	for _, source := range context.sources {
		for _, declaration := range source.file.Decls {
			switch typed := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range typed.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if !ok {
						continue
					}
					switch {
					case source.relativePath == config.contractPath && typeSpec.Name.Name == contactRequestSnapshotType:
						if context.snapshotType != nil {
							return contactRequestScanContext{}, false
						}
						context.snapshotType = typeSpec
						context.allowedNodes[typeSpec] = true
					case source.relativePath == config.contractPath && typeSpec.Name.Name == contactRequestSnapshotReader:
						if context.readerType != nil {
							return contactRequestScanContext{}, false
						}
						context.readerType = typeSpec
						context.allowedNodes[typeSpec] = true
					case source.relativePath == config.optionsPath && typeSpec.Name.Name == "OptionsRepository":
						if context.optionsRepositoryType != nil {
							return contactRequestScanContext{}, false
						}
						context.optionsRepositoryType = typeSpec
					}
				}
			case *ast.FuncDecl:
				if source.relativePath == config.repositoryPath && typed.Name.Name == contactRequestSnapshotMethod && contactRequestReceiverNamesOptionsRepository(typed) {
					if context.repositoryImplementation != nil {
						return contactRequestScanContext{}, false
					}
					context.repositoryImplementation = typed
					context.allowedNodes[typed] = true
				}
				if source.relativePath == config.addContactPath && typed.Recv == nil && typed.Name.Name == "AddContactRequest" {
					if context.addContactRequestFunction != nil {
						return contactRequestScanContext{}, false
					}
					context.addContactRequestFunction = typed
				}
			}
		}
	}

	for _, source := range context.sources {
		if source.relativePath != config.repositoryPath {
			continue
		}
		bindings := contactRequestImports(source.file, config)
		for _, declaration := range source.file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, specification := range general.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if ok && isExactContactRequestRepositoryAssertion(value, source, bindings, context) {
					context.allowedNodes[value] = true
				}
			}
		}
	}
	return context, true
}

func contactRequestPackageIdentityForSource(source contactRequestParsedSource) contactRequestPackageIdentity {
	return contactRequestPackageIdentity{
		path: pathpkg.Dir(filepath.ToSlash(source.relativePath)),
		name: source.file.Name.Name,
	}
}

func contactRequestAnchorsAreComplete(context contactRequestScanContext) bool {
	return context.snapshotType != nil &&
		context.readerType != nil && contactRequestReaderDeclaresExpectedMethod(context.readerType) &&
		context.optionsRepositoryType != nil && contactRequestRepositoryImplementationIsCanonical(context) &&
		context.addContactRequestFunction != nil
}

func contactRequestRepositoryImplementationIsCanonical(context contactRequestScanContext) bool {
	if context.repositoryImplementation == nil || context.repositoryImplementation.Recv == nil || len(context.repositoryImplementation.Recv.List) != 1 {
		return false
	}
	for _, source := range context.sources {
		if source.relativePath != context.config.repositoryPath {
			continue
		}
		bindings := contactRequestImports(source.file, context.config)
		return expressionIsOwnedOptionsRepository(context.repositoryImplementation.Recv.List[0].Type, source, bindings, context, newContactRequestOwnershipResolution())
	}
	return false
}

func contactRequestReaderDeclaresExpectedMethod(reader *ast.TypeSpec) bool {
	interfaceType, ok := reader.Type.(*ast.InterfaceType)
	if !ok || interfaceType.Methods == nil {
		return false
	}
	count := 0
	for _, method := range interfaceType.Methods.List {
		for _, name := range method.Names {
			if name.Name == contactRequestSnapshotMethod {
				count++
			}
		}
	}
	return count == 1
}

func contactRequestReceiverNamesOptionsRepository(function *ast.FuncDecl) bool {
	if function.Recv == nil || len(function.Recv.List) != 1 {
		return false
	}
	return contactRequestTypeExpressionNamesOptionsRepository(function.Recv.List[0].Type)
}

func contactRequestTypeExpressionNamesOptionsRepository(expression ast.Expr) bool {
	switch typed := expression.(type) {
	case *ast.ParenExpr:
		return contactRequestTypeExpressionNamesOptionsRepository(typed.X)
	case *ast.StarExpr:
		return contactRequestTypeExpressionNamesOptionsRepository(typed.X)
	case *ast.Ident:
		return typed.Name == "OptionsRepository"
	}
	return false
}

// This scanner covers statically resolvable Go bindings. Reflection and
// string-built symbol lookup are intentionally outside its evidence boundary.
func countContactRequestProductionReferences(sources []contactRequestSource, config contactRequestScannerConfig) (int, bool) {
	context, ok := prepareContactRequestScanContext(sources, config)
	if !ok {
		return 0, false
	}
	return countContactRequestReferences(context), true
}

func countContactRequestReferences(context contactRequestScanContext) int {
	count := 0
	for _, source := range context.sources {
		count += countContactRequestReferencesInFile(source, context)
	}
	return count
}

func countContactRequestReferencesInFile(source contactRequestParsedSource, context contactRequestScanContext) int {
	file := source.file
	bindings := contactRequestImports(file, context.config)
	selectorIdentifiers := map[*ast.Ident]bool{}
	declarationIdentifiers := map[*ast.Ident]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if context.allowedNodes[node] {
			return false
		}
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			selector := typed
			selectorIdentifiers[selector.Sel] = true
		case *ast.ImportSpec:
			if typed.Name != nil {
				declarationIdentifiers[typed.Name] = true
			}
		case *ast.TypeSpec:
			declarationIdentifiers[typed.Name] = true
		case *ast.FuncDecl:
			declarationIdentifiers[typed.Name] = true
		case *ast.Field:
			for _, name := range typed.Names {
				declarationIdentifiers[name] = true
			}
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				declarationIdentifiers[name] = true
			}
		case *ast.AssignStmt:
			if typed.Tok == token.DEFINE {
				for _, left := range typed.Lhs {
					if name, ok := left.(*ast.Ident); ok {
						declarationIdentifiers[name] = true
					}
				}
			}
		case *ast.RangeStmt:
			if typed.Tok == token.DEFINE {
				if name, ok := typed.Key.(*ast.Ident); ok {
					declarationIdentifiers[name] = true
				}
				if name, ok := typed.Value.(*ast.Ident); ok {
					declarationIdentifiers[name] = true
				}
			}
		}
		return true
	})

	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		if context.allowedNodes[node] {
			return false
		}
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			if selectorUsesCanonicalData(typed, bindings) ||
				selectorUsesCanonicalContactRequestReader(typed, source, bindings, context) ||
				selectorUsesOwnedOptionsRepository(typed, source, bindings, context) {
				count++
			}
		case *ast.Ident:
			if identifierUsesCanonicalContactRequestType(typed, source, bindings, context, selectorIdentifiers, declarationIdentifiers) {
				count++
			}
		}
		return true
	})
	return count
}

func contactRequestImports(file *ast.File, config contactRequestScannerConfig) contactRequestImportBindings {
	bindings := contactRequestImportBindings{
		dataAliases:       map[string]bool{},
		repositoryAliases: map[string]bool{},
	}
	for _, imported := range file.Imports {
		importPath, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			continue
		}
		alias := pathpkg.Base(importPath)
		if imported.Name != nil {
			alias = imported.Name.Name
		}
		if importPath == config.dataImportPath {
			if alias == "." {
				bindings.dotData = true
			} else if alias != "_" {
				bindings.dataAliases[alias] = true
			}
		}
		if importPath == config.repositoryImportPath && alias != "." && alias != "_" {
			bindings.repositoryAliases[alias] = true
		}
	}
	return bindings
}

func selectorUsesCanonicalData(selector *ast.SelectorExpr, bindings contactRequestImportBindings) bool {
	if !isProtectedContactRequestType(selector.Sel.Name) {
		return false
	}
	alias, ok := selector.X.(*ast.Ident)
	return ok && bindings.dataAliases[alias.Name] && (alias.Obj == nil || alias.Obj.Kind == ast.Pkg)
}

func isProtectedContactRequestType(name string) bool {
	return name == contactRequestSnapshotType || name == contactRequestSnapshotReader
}

func identifierUsesCanonicalContactRequestType(identifier *ast.Ident, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext, selectorIdentifiers, declarationIdentifiers map[*ast.Ident]bool) bool {
	if selectorIdentifiers[identifier] || declarationIdentifiers[identifier] || !isProtectedContactRequestType(identifier.Name) {
		return false
	}
	if identifier.Obj == nil && bindings.dotData {
		return true
	}
	if !contactRequestSourceIsPackage(source, "models/data", "data") {
		return false
	}
	wanted := context.snapshotType
	if identifier.Name == contactRequestSnapshotReader {
		wanted = context.readerType
	}
	if wanted == nil {
		return false
	}
	return identifier.Obj == nil || identifier.Obj.Decl == wanted
}

func contactRequestSourceIsPackage(source contactRequestParsedSource, directory, packageName string) bool {
	return pathpkg.Dir(filepath.ToSlash(source.relativePath)) == directory && source.file.Name.Name == packageName
}

func selectorUsesCanonicalContactRequestReader(selector *ast.SelectorExpr, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext) bool {
	if selector.Sel.Name != contactRequestSnapshotMethod {
		return false
	}
	return expressionIsCanonicalContactRequestReader(selector.X, source, bindings, context, map[*ast.Object]bool{})
}

func expressionIsCanonicalContactRequestReader(expression ast.Expr, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext, seen map[*ast.Object]bool) bool {
	switch typed := expression.(type) {
	case *ast.ParenExpr:
		return expressionIsCanonicalContactRequestReader(typed.X, source, bindings, context, seen)
	case *ast.StarExpr:
		return expressionIsCanonicalContactRequestReader(typed.X, source, bindings, context, seen)
	case *ast.SelectorExpr:
		alias, ok := typed.X.(*ast.Ident)
		return ok && typed.Sel.Name == contactRequestSnapshotReader && bindings.dataAliases[alias.Name] && (alias.Obj == nil || alias.Obj.Kind == ast.Pkg)
	case *ast.Ident:
		if typed.Name == contactRequestSnapshotReader {
			if typed.Obj == nil && bindings.dotData {
				return true
			}
			if contactRequestSourceIsPackage(source, "models/data", "data") && context.readerType != nil && (typed.Obj == nil || typed.Obj.Decl == context.readerType) {
				return true
			}
		}
		if typed.Obj == nil || seen[typed.Obj] {
			return false
		}
		seen[typed.Obj] = true
		return objectDeclaresCanonicalContactRequestReader(typed.Obj.Decl, typed, source, bindings, context, seen)
	}
	return false
}

func objectDeclaresCanonicalContactRequestReader(declaration any, identifier *ast.Ident, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext, seen map[*ast.Object]bool) bool {
	switch typed := declaration.(type) {
	case *ast.Field:
		return expressionIsCanonicalContactRequestReader(typed.Type, source, bindings, context, seen)
	case *ast.ValueSpec:
		if typed.Type != nil {
			return expressionIsCanonicalContactRequestReader(typed.Type, source, bindings, context, seen)
		}
		for index, name := range typed.Names {
			if name.Obj != identifier.Obj || len(typed.Values) == 0 {
				continue
			}
			valueIndex := index
			if valueIndex >= len(typed.Values) {
				valueIndex = len(typed.Values) - 1
			}
			return expressionIsCanonicalContactRequestReader(typed.Values[valueIndex], source, bindings, context, seen)
		}
	case *ast.AssignStmt:
		for index, left := range typed.Lhs {
			name, ok := left.(*ast.Ident)
			if !ok || name.Obj != identifier.Obj || len(typed.Rhs) == 0 {
				continue
			}
			valueIndex := index
			if valueIndex >= len(typed.Rhs) {
				valueIndex = len(typed.Rhs) - 1
			}
			return expressionIsCanonicalContactRequestReader(typed.Rhs[valueIndex], source, bindings, context, seen)
		}
	}
	return false
}

func selectorUsesOwnedOptionsRepository(selector *ast.SelectorExpr, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext) bool {
	if selector.Sel.Name != contactRequestSnapshotMethod {
		return false
	}
	return expressionIsOwnedOptionsRepository(selector.X, source, bindings, context, newContactRequestOwnershipResolution())
}

func newContactRequestOwnershipResolution() *contactRequestOwnershipResolution {
	return &contactRequestOwnershipResolution{
		objects: map[*ast.Object]bool{},
		symbols: map[contactRequestPackageSymbol]bool{},
	}
}

func expressionIsOwnedOptionsRepository(expression ast.Expr, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext, seen *contactRequestOwnershipResolution) bool {
	switch typed := expression.(type) {
	case *ast.ParenExpr:
		return expressionIsOwnedOptionsRepository(typed.X, source, bindings, context, seen)
	case *ast.StarExpr:
		return expressionIsOwnedOptionsRepository(typed.X, source, bindings, context, seen)
	case *ast.UnaryExpr:
		return typed.Op == token.AND && expressionIsOwnedOptionsRepository(typed.X, source, bindings, context, seen)
	case *ast.CompositeLit:
		return expressionIsOwnedOptionsRepository(typed.Type, source, bindings, context, seen)
	case *ast.SelectorExpr:
		alias, ok := typed.X.(*ast.Ident)
		return ok && typed.Sel.Name == "OptionsRepository" && bindings.repositoryAliases[alias.Name] && (alias.Obj == nil || alias.Obj.Kind == ast.Pkg)
	case *ast.CallExpr:
		if expressionIsOwnedOptionsRepository(typed.Fun, source, bindings, context, seen) {
			return true
		}
		if identifier, ok := typed.Fun.(*ast.Ident); ok {
			if identifier.Name == "new" && len(typed.Args) == 1 {
				return expressionIsOwnedOptionsRepository(typed.Args[0], source, bindings, context, seen)
			}
			if identifier.Name == "NewOptionsRepository" && identifier.Obj == nil && contactRequestSourceIsPackage(source, "database/postgres", "postgres") && context.optionsRepositoryType != nil {
				return true
			}
		}
		if selector, ok := typed.Fun.(*ast.SelectorExpr); ok {
			alias, aliasOK := selector.X.(*ast.Ident)
			if aliasOK && selector.Sel.Name == "NewOptionsRepository" && bindings.repositoryAliases[alias.Name] && (alias.Obj == nil || alias.Obj.Kind == ast.Pkg) {
				return true
			}
		}
	case *ast.Ident:
		if typed.Name == "OptionsRepository" && contactRequestSourceIsPackage(source, "database/postgres", "postgres") && context.optionsRepositoryType != nil {
			if typed.Obj == nil || typed.Obj.Decl == context.optionsRepositoryType {
				return true
			}
		}
		if typed.Obj == nil {
			return packageTypeDeclaresOwnedOptionsRepository(typed.Name, source, context, seen)
		}
		if seen.objects[typed.Obj] {
			return false
		}
		seen.objects[typed.Obj] = true
		return objectDeclaresOwnedOptionsRepository(typed.Obj.Decl, typed, source, bindings, context, seen)
	}
	return false
}

func objectDeclaresOwnedOptionsRepository(declaration any, identifier *ast.Ident, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext, seen *contactRequestOwnershipResolution) bool {
	switch typed := declaration.(type) {
	case *ast.TypeSpec:
		if !typed.Assign.IsValid() {
			return false
		}
		return expressionIsOwnedOptionsRepository(typed.Type, source, bindings, context, seen)
	case *ast.Field:
		return expressionIsOwnedOptionsRepository(typed.Type, source, bindings, context, seen)
	case *ast.ValueSpec:
		if typed.Type != nil && expressionIsOwnedOptionsRepository(typed.Type, source, bindings, context, seen) {
			return true
		}
		for index, name := range typed.Names {
			if name.Obj != identifier.Obj || len(typed.Values) == 0 {
				continue
			}
			valueIndex := index
			if valueIndex >= len(typed.Values) {
				valueIndex = len(typed.Values) - 1
			}
			return expressionIsOwnedOptionsRepository(typed.Values[valueIndex], source, bindings, context, seen)
		}
	case *ast.AssignStmt:
		for index, left := range typed.Lhs {
			name, ok := left.(*ast.Ident)
			if !ok || name.Obj != identifier.Obj || len(typed.Rhs) == 0 {
				continue
			}
			valueIndex := index
			if valueIndex >= len(typed.Rhs) {
				valueIndex = len(typed.Rhs) - 1
			}
			return expressionIsOwnedOptionsRepository(typed.Rhs[valueIndex], source, bindings, context, seen)
		}
	}
	return false
}

func packageTypeDeclaresOwnedOptionsRepository(name string, source contactRequestParsedSource, context contactRequestScanContext, seen *contactRequestOwnershipResolution) bool {
	identity := contactRequestPackageIdentityForSource(source)
	symbol := contactRequestPackageSymbol{packageIdentity: identity, name: name}
	if seen.symbols[symbol] {
		return false
	}
	declarations := context.packageTypes[identity][name]
	if len(declarations) != 1 || !declarations[0].declaration.Assign.IsValid() {
		return false
	}
	seen.symbols[symbol] = true
	declaration := declarations[0]
	bindings := contactRequestImports(declaration.source.file, context.config)
	return expressionIsOwnedOptionsRepository(declaration.declaration.Type, declaration.source, bindings, context, seen)
}

func isExactContactRequestRepositoryAssertion(value *ast.ValueSpec, source contactRequestParsedSource, bindings contactRequestImportBindings, context contactRequestScanContext) bool {
	if len(value.Names) != 1 || value.Names[0].Name != "_" || len(value.Values) != 1 {
		return false
	}
	reader, ok := value.Type.(*ast.SelectorExpr)
	if !ok || reader.Sel.Name != contactRequestSnapshotReader || !selectorUsesCanonicalData(reader, bindings) {
		return false
	}
	conversion, ok := value.Values[0].(*ast.CallExpr)
	if !ok || len(conversion.Args) != 1 {
		return false
	}
	nilValue, ok := conversion.Args[0].(*ast.Ident)
	if !ok || nilValue.Name != "nil" || nilValue.Obj != nil {
		return false
	}
	parenthesized, ok := conversion.Fun.(*ast.ParenExpr)
	if !ok {
		return false
	}
	pointer, ok := parenthesized.X.(*ast.StarExpr)
	return ok && expressionIsOwnedOptionsRepository(pointer.X, source, bindings, context, newContactRequestOwnershipResolution())
}

func parseContactRequestFixture(source string) (*ast.File, bool) {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0)
	return file, err == nil
}

func contactRequestWorkspaceRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contact-request snapshot test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
