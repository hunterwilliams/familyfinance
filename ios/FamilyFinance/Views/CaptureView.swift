import SwiftUI
import PhotosUI

struct CaptureView: View {
    @Environment(\.modelContext) private var modelContext
    @Environment(\.dismiss) private var dismiss

    @State private var selectedItem: PhotosPickerItem?
    @State private var selectedImage: UIImage?
    @State private var isProcessing = false

    var body: some View {
        VStack(spacing: 20) {
            Text("Select a receipt screenshot or notification image. Nothing is imported automatically.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding()

            PhotosPicker(selection: $selectedItem, matching: .images) {
                Label("Choose from Photos", systemImage: "photo")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .padding(.horizontal)

            if let image = selectedImage {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFit()
                    .frame(maxHeight: 200)
                    .padding()
            }

            if isProcessing {
                ProgressView("Reading receipt...")
            }
        }
        .navigationTitle("Capture")
        .onChange(of: selectedItem) { _, newItem in
            Task { await loadImage(from: newItem) }
        }
    }

    private func loadImage(from item: PhotosPickerItem?) async {
        guard let item,
              let data = try? await item.loadTransferable(type: Data.self),
              let image = UIImage(data: data) else { return }

        selectedImage = image
        isProcessing = true
        let ocr = await OCRService.parseReceipt(image: image)

        let members = (try? await APIClient.shared.fetchMembers()) ?? []
        let accounts = (try? await APIClient.shared.fetchAccounts()) ?? []

        let draft = TransactionDraft(
            amountSatang: ocr.amountSatang ?? 0,
            transactionDate: ocr.date ?? .now,
            descriptionText: ocr.merchant ?? "",
            paidByMemberID: members.first?.id ?? "",
            accountID: accounts.first?.id ?? "",
            receiptImageData: data
        )
        modelContext.insert(draft)
        try? modelContext.save()
        isProcessing = false
        dismiss()
    }
}
