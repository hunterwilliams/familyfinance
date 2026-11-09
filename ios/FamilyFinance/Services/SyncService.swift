import Foundation
import SwiftData

enum SyncService {
    static let shared = SyncServiceImpl()
}

final class SyncServiceImpl {
    func syncPending(context: ModelContext) async {
        let descriptor = FetchDescriptor<TransactionDraft>(
            predicate: #Predicate { !$0.synced }
        )
        guard let drafts = try? context.fetch(descriptor) else { return }

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

        for draft in drafts {
            do {
                try await APIClient.shared.createTransaction(CreateTransactionRequest(
                    account_id: draft.accountID,
                    category_id: draft.categoryID,
                    paid_by_member_id: draft.paidByMemberID,
                    transaction_date: formatter.string(from: draft.transactionDate),
                    description: draft.descriptionText,
                    amount_satang: draft.amountSatang
                ))
                draft.synced = true
            } catch {
                continue
            }
        }
        try? context.save()
    }
}
