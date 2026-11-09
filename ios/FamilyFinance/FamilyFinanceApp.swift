import SwiftUI
import SwiftData

@main
struct FamilyFinanceApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
        }
        .modelContainer(for: TransactionDraft.self)
    }
}
