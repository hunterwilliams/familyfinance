import SwiftUI
import SwiftData

struct ConfirmTransactionView: View {
    @Bindable var draft: TransactionDraft
    @Environment(\.modelContext) private var modelContext
    @Environment(\.dismiss) private var dismiss

    @State private var members: [MemberDTO] = []
    @State private var accounts: [AccountDTO] = []
    @State private var categories: [CategoryDTO] = []
    @State private var amountTHB: String = ""
    @State private var saving = false

    var body: some View {
        Form {
            Section("Review before saving") {
                TextField("Description", text: $draft.descriptionText)
                TextField("Amount (THB)", text: $amountTHB)
                    .keyboardType(.decimalPad)
                DatePicker("Date", selection: $draft.transactionDate, displayedComponents: .date)

                Picker("Paid by", selection: $draft.paidByMemberID) {
                    ForEach(members) { m in
                        Text(m.name).tag(m.id)
                    }
                }

                Picker("Account", selection: $draft.accountID) {
                    ForEach(accounts) { a in
                        Text(a.name).tag(a.id)
                    }
                }

                Picker("Category", selection: Binding(
                    get: { draft.categoryID ?? "" },
                    set: { draft.categoryID = $0.isEmpty ? nil : $0 }
                )) {
                    Text("None").tag("")
                    ForEach(categories) { c in
                        Text(c.name).tag(c.id)
                    }
                }
            }

            Section {
                Button("Save to server") {
                    Task { await save() }
                }
                .disabled(saving || draft.paidByMemberID.isEmpty)

                Button("Discard", role: .destructive) {
                    modelContext.delete(draft)
                    try? modelContext.save()
                    dismiss()
                }
            }
        }
        .navigationTitle("Confirm")
        .task {
            members = (try? await APIClient.shared.fetchMembers()) ?? []
            accounts = (try? await APIClient.shared.fetchAccounts()) ?? []
            categories = (try? await APIClient.shared.fetchCategories()) ?? []
            amountTHB = String(format: "%.2f", Double(draft.amountSatang) / 100)
        }
        .onChange(of: amountTHB) { _, newValue in
            if let value = Double(newValue) {
                draft.amountSatang = Int(value * 100)
            }
        }
    }

    private func save() async {
        saving = true
        defer { saving = false }

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"

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
            try? modelContext.save()
            dismiss()
        } catch {
            // stays on confirm screen for retry
        }
    }
}
