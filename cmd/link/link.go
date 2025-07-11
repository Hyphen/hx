package link

import (
	"errors"
	"fmt"

	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/internal/zelda"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	qr      bool
	domain  string
	tags    []string
	code    string
	title   string
	printer *cprint.CPrinter
)

var LinkCmd = &cobra.Command{
	Use:   "link <long_url>",
	Short: "Create a shortened URL",
	Long: `
The link command creates a shortened URL for a given long URL.

Usage:
  hyphen link <long_url> [flags]

This command allows you to:
- Generate a short URL for a given long URL
- Optionally create a QR code for the shortened URL
- Specify a custom domain, tags, custom short code, and title
- Retrieve the shortened URL and optional QR code URL

The command will automatically add 'https://' to the long URL if not present.

If no custom domain is specified, the command will use the first ready domain 
associated with your organization.

Examples:
  hyphen link example.com
  hyphen link https://very-long-url.com --qr
  hyphen link example.com --tag promo --tag summer2023 --code summer-sale
  hyphen link example.com --title "Summer Sale" --domain custom.short.link

Use 'hyphen link --help' for more information about available flags.
`,
	Args: cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		service := newService(zelda.NewService())

		orgId, err := flags.GetOrganizationID()
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to get organization ID: %w", err))
			return
		}

		if flags.VerboseFlag {
			printer.Info("Fetching domain information...")
		}

		domain, err := service.GetDomain(orgId)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to get domain: %w", err))
			return
		}

		if flags.VerboseFlag {
			printer.Success(fmt.Sprintf("Using domain: %s", domain))
		}

		var codePtr, titlePtr *string
		if code != "" {
			codePtr = &code
		}
		if title != "" {
			titlePtr = &title
		}

		longURL := args[0]
		// add https:// if longURL does not have it
		if longURL[:8] != "https://" {
			longURL = "https://" + longURL
		}

		newCode := zelda.Code{
			LongURL:        longURL,
			Domain:         domain,
			Code:           codePtr,
			Title:          titlePtr,
			Tags:           tags,
			OrganizationID: orgId,
		}

		if flags.VerboseFlag {
			printer.Info("Generating short code...")
		}

		shortCode, err := service.GenerateShortCode(orgId, newCode)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to generate short code: %w", err))
			return
		}

		var qrCodeURL string
		if qr == true {
			if flags.VerboseFlag {
				printer.Info("Generating QR code...")
			}
			qrCode, err := service.GenerateQR(orgId, *shortCode.ID)
			if err != nil {
				printer.Error(cmd, fmt.Errorf("failed to generate QR code: %w", err))
				return
			}
			qrCodeURL = qrCode.QRLink
		}

		if flags.VerboseFlag {
			printer.Success("Link generation successful")
		}

		shortURL := fmt.Sprintf("%s/%s", domain, *shortCode.Code)

		if flags.VerboseFlag {
			printer.PrintDetail("Long URL", args[0])
			printer.PrintDetail("Short URL", shortURL)
			printer.PrintDetail("Short Code", *shortCode.Code)
			if shortCode.Title != nil {
				printer.PrintDetail("Title", *shortCode.Title)
			}
			if len(tags) > 0 {
				printer.PrintDetail("Tags", fmt.Sprintf("%v", tags))
			}
			if qrCodeURL != "" {
				printer.PrintDetail("QR Code URL", qrCodeURL)
			}

			printer.GreenPrint("\nURL shortened successfully!")

			if qr {
				printer.PrintHeader("QR Code")
				printer.PrintNorm(shortURL)
			}
		} else {
			printer.Print(shortURL)
			if qr {
				printer.Print(qrCodeURL)
			}
		}
	},
}

func init() {
	LinkCmd.Flags().BoolVar(&qr, "qr", false, "Generate a QR code")
	LinkCmd.Flags().StringVar(&domain, "domain", "", "The domain to use when shortening")
	LinkCmd.Flags().StringArrayVar(&tags, "tag", []string{}, "Tags for the shortened URL. Can be specified multiple times")
	LinkCmd.Flags().StringVar(&code, "code", "", "Custom short code")
	LinkCmd.Flags().StringVar(&title, "title", "", "Title for the shortened URL")
}

type service struct {
	zeldaService zelda.ZeldaServicer
}

func newService(zeldaService zelda.ZeldaServicer) *service {
	return &service{
		zeldaService,
	}
}

func (s *service) GenerateShortCode(orgID string, code zelda.Code) (zelda.Code, error) {
	return s.zeldaService.CreateCode(orgID, code)
}

func (s *service) GetDomain(organizationId string) (string, error) {
	if domain != "" {
		return domain, nil
	}

	domains, err := s.zeldaService.ListDomains(organizationId, 100, 1)
	if err != nil {
		return "", err
	}

	if domains == nil || len(domains) == 0 {
		return "", errors.New("No domains found")
	}
	for _, domain := range domains {
		if domain.Status == "ready" {
			return domain.Domain, nil
		}
	}
	return "", errors.New("No Domain found with status ready")

}

func (s *service) GenerateQR(organizationID, codeId string) (zelda.QR, error) {
	return s.zeldaService.CreateQRCode(organizationID, codeId)
}
