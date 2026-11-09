import Foundation
import SwiftData

@Model
final class TransactionDraft {
    var id: UUID
    var amountSatang: Int
    var transactionDate: Date
    var descriptionText: String
    var paidByMemberID: String
    var accountID: String
    var categoryID: String?
    var receiptImageData: Data?
    var synced: Bool
    var createdAt: Date

    init(
        amountSatang: Int = 0,
        transactionDate: Date = .now,
        descriptionText: String = "",
        paidByMemberID: String = "",
        accountID: String = "",
        categoryID: String? = nil,
        receiptImageData: Data? = nil,
        synced: Bool = false
    ) {
        self.id = UUID()
        self.amountSatang = amountSatang
        self.transactionDate = transactionDate
        self.descriptionText = descriptionText
        self.paidByMemberID = paidByMemberID
        self.accountID = accountID
        self.categoryID = categoryID
        self.receiptImageData = receiptImageData
        self.synced = synced
        self.createdAt = .now
    }
}
