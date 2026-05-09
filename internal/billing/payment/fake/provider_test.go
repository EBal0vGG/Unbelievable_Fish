package fake

import (
	"context"
	"strings"
	"testing"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
)

func TestProvider_CreateDealInvoice_PaymentURLUsesFakeConfirmRoute(t *testing.T) {
	ctx := context.Background()
	var p Provider
	resp, err := p.CreateDealInvoice(ctx, billingapp.CreateDealInvoiceRequest{
		InvoiceID: "inv-abc",
		ReturnURL: "http://localhost:8085",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.PaymentURL, "/billing/invoices/inv-abc/fake-confirm") {
		t.Fatalf("got %q", resp.PaymentURL)
	}
}
