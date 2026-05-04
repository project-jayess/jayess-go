package binding

const RuntimeHeader = "jayess_runtime.h"

type OwnershipRule string

const (
	RuntimeOwnsReturnedValues  OwnershipRule = "runtime-owns-returned-values"
	BorrowedViewsDuringCall    OwnershipRule = "borrowed-views-during-call"
	CopiedStringsForStorage    OwnershipRule = "copied-strings-for-storage"
	CopiedBytesForStorage      OwnershipRule = "copied-bytes-for-storage"
	ManagedHandlesClosable     OwnershipRule = "managed-handles-closable"
	NoDoubleFreeAcrossNative   OwnershipRule = "no-double-free-across-native"
	NoUseAfterFreeAcrossNative OwnershipRule = "no-use-after-free-across-native"
)

func OwnershipRules() []OwnershipRule {
	return []OwnershipRule{
		RuntimeOwnsReturnedValues,
		BorrowedViewsDuringCall,
		CopiedStringsForStorage,
		CopiedBytesForStorage,
		ManagedHandlesClosable,
		NoDoubleFreeAcrossNative,
		NoUseAfterFreeAcrossNative,
	}
}
