package main

import (
	"log"
	"os"

	"github.com/gosuri/uilive"
	"github.com/olekukonko/tablewriter"

	"time"

	jhandacommands "github.com/pivotal-cf/jhanda/commands"
	"github.com/pivotal-cf/jhanda/flags"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/extractor"
	"github.com/pivotal-cf/om/formcontent"
	"github.com/pivotal-cf/om/network"
	"github.com/pivotal-cf/om/presenters"
	"github.com/pivotal-cf/om/progress"
)

var version = "unknown"

const applySleepSeconds = 10

func main() {
	liveWriter := uilive.New()

	stdout := log.New(os.Stdout, "", 0)
	stderr := log.New(os.Stderr, "", 0)

	var global struct {
		ClientID          string `short:"c" long:"client-id"                           description:"Client ID for the Ops Manager VM (not required for unauthenticated commands)"`
		ClientSecret      string `short:"s" long:"client-secret"                       description:"Client Secret for the Ops Manager VM (not required for unauthenticated commands)"`
		Format            string `short:"f" long:"format"              default:"table" description:"Format to print as (options: table,json)"`
		Help              bool   `short:"h" long:"help"                default:"false" description:"prints this usage information"`
		Password          string `short:"p" long:"password"                            description:"admin password for the Ops Manager VM (not required for unauthenticated commands)"`
		RequestTimeout    int    `short:"r" long:"request-timeout"     default:"1800"  description:"timeout in seconds for HTTP requests to Ops Manager"`
		SkipSSLValidation bool   `short:"k" long:"skip-ssl-validation" default:"false" description:"skip ssl certificate validation during http requests"`
		Target            string `short:"t" long:"target"                              description:"location of the Ops Manager VM"`
		Username          string `short:"u" long:"username"                            description:"admin username for the Ops Manager VM (not required for unauthenticated commands)"`
		Version           bool   `short:"v" long:"version"             default:"false" description:"prints the om release version"`
	}

	args, err := flags.Parse(&global, os.Args[1:])
	if err != nil {
		stdout.Fatal(err)
	}

	globalFlagsUsage, err := flags.Usage(global)
	if err != nil {
		stdout.Fatal(err)
	}

	var command string
	if len(args) > 0 {
		command, args = args[0], args[1:]
	}

	if global.Version {
		command = "version"
	}

	if global.Help {
		command = "help"
	}

	if command == "" {
		command = "help"
	}

	if global.Username == "" {
		global.Username = os.Getenv("OM_USERNAME")
	}

	if global.Password == "" {
		global.Password = os.Getenv("OM_PASSWORD")
	}

	if global.ClientID == "" {
		global.ClientID = os.Getenv("OM_CLIENT_ID")
	}

	if global.ClientSecret == "" {
		global.ClientSecret = os.Getenv("OM_CLIENT_SECRET")
	}

	requestTimeout := time.Duration(global.RequestTimeout) * time.Second

	unauthenticatedClient := network.NewUnauthenticatedClient(global.Target, global.SkipSSLValidation, requestTimeout)

	authedClient, err := network.NewOAuthClient(global.Target, global.Username, global.Password, global.ClientID, global.ClientSecret, global.SkipSSLValidation, false, requestTimeout)
	if err != nil {
		stdout.Fatal(err)
	}

	authedCookieClient, err := network.NewOAuthClient(global.Target, global.Username, global.Password, global.ClientID, global.ClientSecret, global.SkipSSLValidation, true, requestTimeout)
	if err != nil {
		stdout.Fatal(err)
	}

	setupService := api.NewSetupService(unauthenticatedClient)
	uploadStemcellService := api.NewUploadStemcellService(authedClient, progress.NewBar())
	stagedProductsService := api.NewStagedProductsService(authedClient)
	deployedProductsService := api.NewDeployedProductsService(authedClient)
	credentialReferencesService := api.NewCredentialReferencesService(authedClient, progress.NewBar())
	credentialsService := api.NewCredentialsService(authedClient, progress.NewBar())
	availableProductsService := api.NewAvailableProductsService(authedClient, progress.NewBar(), liveWriter)
	diagnosticService := api.NewDiagnosticService(authedClient)
	importInstallationService := api.NewInstallationAssetService(unauthenticatedClient, progress.NewBar(), liveWriter)
	exportInstallationService := api.NewInstallationAssetService(authedClient, progress.NewBar(), liveWriter)
	deleteInstallationService := api.NewInstallationAssetService(authedClient, nil, nil)
	installationsService := api.NewInstallationsService(authedClient)
	errandsService := api.NewErrandsService(authedClient)
	pendingChangesService := api.NewPendingChangesService(authedClient)
	logWriter := commands.NewLogWriter(os.Stdout)
	tableWriter := tablewriter.NewWriter(os.Stdout)
	requestService := api.NewRequestService(authedClient)
	jobsService := api.NewJobsService(authedClient)
	boshService := api.NewBoshFormService(authedCookieClient)
	dashboardService := api.NewDashboardService(authedCookieClient)
	certificateAuthoritiesService := api.NewCertificateAuthoritiesService(authedClient)
	certificatesService := api.NewCertificatesService(authedClient)
	directorService := api.NewDirectorService(authedClient)

	form, err := formcontent.NewForm()
	if err != nil {
		stdout.Fatal(err)
	}

	extractor := extractor.ProductUnzipper{}

	var presenter presenters.Presenter
	switch global.Format {
	case "table":
		presenter = presenters.NewTablePresenter(tableWriter)
	case "json":
		presenter = presenters.NewJSONPresenter(os.Stdout)
	default:
		stdout.Fatal("Format not supported")
	}

	commandSet := jhandacommands.Set{}
	commandSet["help"] = commands.NewHelp(os.Stdout, globalFlagsUsage, commandSet)
	commandSet["version"] = commands.NewVersion(version, os.Stdout)
	commandSet["configure-authentication"] = commands.NewConfigureAuthentication(setupService, stdout)
	commandSet["configure-bosh"] = commands.NewConfigureBosh(boshService, diagnosticService, stdout)
	commandSet["revert-staged-changes"] = commands.NewRevertStagedChanges(dashboardService, stdout)
	commandSet["upload-stemcell"] = commands.NewUploadStemcell(form, uploadStemcellService, diagnosticService, stdout)
	commandSet["upload-product"] = commands.NewUploadProduct(form, extractor, availableProductsService, stdout)
	commandSet["delete-unused-products"] = commands.NewDeleteUnusedProducts(availableProductsService, stdout)
	commandSet["stage-product"] = commands.NewStageProduct(stagedProductsService, deployedProductsService, availableProductsService, diagnosticService, stdout)
	commandSet["unstage-product"] = commands.NewUnstageProduct(stagedProductsService, stdout)
	commandSet["configure-product"] = commands.NewConfigureProduct(stagedProductsService, jobsService, stdout)
	commandSet["export-installation"] = commands.NewExportInstallation(exportInstallationService, stdout)
	commandSet["import-installation"] = commands.NewImportInstallation(form, importInstallationService, setupService, stdout)
	commandSet["delete-installation"] = commands.NewDeleteInstallation(deleteInstallationService, installationsService, logWriter, stdout, applySleepSeconds)
	commandSet["apply-changes"] = commands.NewApplyChanges(installationsService, logWriter, stdout, applySleepSeconds)
	commandSet["curl"] = commands.NewCurl(requestService, stdout, stderr)
	commandSet["available-products"] = commands.NewAvailableProducts(availableProductsService, presenter, stdout)
	commandSet["errands"] = commands.NewErrands(presenter, errandsService, stagedProductsService)
	commandSet["set-errand-state"] = commands.NewSetErrandState(errandsService, stagedProductsService)
	commandSet["credential-references"] = commands.NewCredentialReferences(credentialReferencesService, deployedProductsService, presenter, stdout)
	commandSet["credentials"] = commands.NewCredentials(credentialsService, deployedProductsService, presenter, stdout)
	commandSet["staged-products"] = commands.NewStagedProducts(presenter, diagnosticService)
	commandSet["deployed-products"] = commands.NewDeployedProducts(presenter, diagnosticService)
	commandSet["delete-product"] = commands.NewDeleteProduct(availableProductsService)
	commandSet["pending-changes"] = commands.NewPendingChanges(presenter, pendingChangesService)
	commandSet["installations"] = commands.NewInstallations(installationsService, presenter)
	commandSet["installation-log"] = commands.NewInstallationLog(installationsService, stdout)
	commandSet["certificate-authorities"] = commands.NewCertificateAuthorities(certificateAuthoritiesService, presenter)
	commandSet["generate-certificate"] = commands.NewGenerateCertificate(certificatesService, stdout)
	commandSet["generate-certificate-authority"] = commands.NewGenerateCertificateAuthority(certificateAuthoritiesService, presenter)
	commandSet["regenerate-certificates"] = commands.NewRegenerateCertificateAuthority(certificateAuthoritiesService, stdout)
	commandSet["create-certificate-authority"] = commands.NewCreateCertificateAuthority(certificateAuthoritiesService, presenter)
	commandSet["activate-certificate-authority"] = commands.NewActivateCertificateAuthority(certificateAuthoritiesService, stdout)
	commandSet["delete-certificate-authority"] = commands.NewDeleteCertificateAuthority(certificateAuthoritiesService, stdout)
	commandSet["configure-director"] = commands.NewConfigureDirector(directorService, stdout)

	err = commandSet.Execute(command, args)
	if err != nil {
		stderr.Fatal(err)
	}
}
