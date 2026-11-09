import SwiftUI
import SwiftData

struct ContentView: View {
    @Environment(\.modelContext) private var modelContext
    @Query(sort: \TransactionDraft.createdAt, order: .reverse) private var drafts: [TransactionDraft]

    var body: some View {
        NavigationStack {
            List {
                Section("Quick capture") {
                    NavigationLink("Add expense") {
                        CaptureView()
                    }
                }

                if !drafts.isEmpty {
                    Section("Pending sync") {
                        ForEach(drafts.filter { !$0.synced }) { draft in
                            NavigationLink(draft.descriptionText) {
                                ConfirmTransactionView(draft: draft)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Family Finance")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Sync") {
                        Task { await SyncService.shared.syncPending(context: modelContext) }
                    }
                }
            }
        }
    }
}
