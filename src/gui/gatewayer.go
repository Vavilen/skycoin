package gui

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/coin"
	"github.com/skycoin/skycoin/src/visor"
	"github.com/skycoin/skycoin/src/visor/historydb"
	"github.com/skycoin/skycoin/src/wallet"
)

// Gatewayer interface for Gateway methods
type Gatewayer interface {
	Spend(wltID string, coins uint64, dest cipher.Address) (*coin.Transaction, error)
	GetWalletBalance(wltID string) (wallet.BalancePair, error)
	GetWallet(wltID string) (wallet.Wallet, error)
	UpdateWalletLabel(wltID, label string) error
	GetAddressTxns(a cipher.Address) (*visor.TransactionResults, error)
	GetUxOutByID(id cipher.SHA256) (*historydb.UxOut, error)
	GetWalletUnconfirmedTxns(wltID string) ([]visor.UnconfirmedTxn, error)
	CreateWallet(wltName string, options wallet.Options) (wallet.Wallet, error)
	ScanAheadWalletAddresses(wltName string, scanN uint64) (wallet.Wallet, error)
	NewAddresses(wltID string, n uint64) ([]cipher.Address, error)
	GetAddressCount() (uint64, error)
}
