package link

import (
	"errors"
	"fmt"

	"github.com/Hyphen/cli/internal/zelda"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	qr     bool
	domain string
	tags   []string
	code   string
	title  string
)

var LinkCmd = &cobra.Command{
	Use:   "link <long_url>",
	Short: "Shorten a URL and optionally generate a QR code",
	Long: `
Shorten a long URL and optionally generate a QR code for the shortened link.

This command takes a long URL as an argument and creates a shortened version using
the Hyphen URL shortening service. It allows customization of the short link and
provides options to add metadata such as tags and a title. Additionally, it can
generate a QR code for easy sharing of the shortened URL.

Usage:
  hyphen link <long_url> [flags]

Arguments:
  <long_url>    The original long URL to be shortened

Flags:
  --qr              Generate a QR code for the shortened URL
  --domain string   Specify a custom domain for the short URL (default: organization's default domain)
  --tag strings     Add tags to the shortened URL (can be used multiple times)
  --code string     Set a custom short code for the URL (if available)
  --title string    Add a title to the shortened URL

The command will display a summary of the shortened URL, including the original long URL,
the new short URL, and any additional metadata or QR code information if applicable.

Examples:
  hyphen link https://example.com/very/long/url
  hyphen link https://example.com/page --title "My Page" --tag promo --tag summer2024 --qr
  hyphen link https://example.com/custom --code mycode --domain custom.com
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		service := newService(zelda.NewService())

		orgId, err := flags.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to get organization ID: %w", err))
			return
		}

		if flags.VerboseFlag {
			cprint.Info("Fetching domain information...")
		}

		domain, err := service.GetDomain(orgId)
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to get domain: %w", err))
			return
		}

		if flags.VerboseFlag {
			cprint.Success(fmt.Sprintf("Using domain: %s", domain))
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
			cprint.Info("Generating short code...")
		}

		shortCode, err := service.GenerateShortCode(newCode)
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to generate short code: %w", err))
			return
		}

		var qrCodeURL string
		if qr == true {
			if flags.VerboseFlag {
				cprint.Info("Generating QR code...")
			}
			qrCode, err := service.GenerateQR(orgId, *shortCode.ID)
			if err != nil {
				cprint.Error(cmd, fmt.Errorf("failed to generate QR code: %w", err))
				return
			}
			qrCodeURL = qrCode.QRLink
		}

		if flags.VerboseFlag {
			cprint.Success("Link generation successful")
		}

		shortURL := fmt.Sprintf("%s/%s", domain, *shortCode.Code)

		if flags.VerboseFlag {
			cprint.PrintDetail("Long URL", args[0])
			cprint.PrintDetail("Short URL", shortURL)
			cprint.PrintDetail("Short Code", *shortCode.Code)
			if shortCode.Title != nil {
				cprint.PrintDetail("Title", *shortCode.Title)
			}
			if len(tags) > 0 {
				cprint.PrintDetail("Tags", fmt.Sprintf("%v", tags))
			}
			if qrCodeURL != "" {
				cprint.PrintDetail("QR Code URL", qrCodeURL)
			}

			cprint.GreenPrint("\nURL shortened successfully!")

			if qr {
				cprint.PrintHeader("QR Code")
				cprint.PrintNorm(shortURL)
			}
		} else {
			cprint.Print(shortURL)
			if qr {
				cprint.Print(qrCodeURL)
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

func (s *service) GenerateShortCode(code zelda.Code) (zelda.Code, error) {
	return s.zeldaService.CreateCode(code)
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
