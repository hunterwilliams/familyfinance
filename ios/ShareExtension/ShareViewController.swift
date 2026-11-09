import UIKit
import Social
import UniformTypeIdentifiers

/// Share Extension entry point — user shares a receipt image from SCB, Grab, etc.
/// Opens confirm flow; never auto-saves.
class ShareViewController: SLComposeServiceViewController {
    override func isContentValid() -> Bool { true }

    override func didSelectPost() {
        // In a full Xcode project, pass extensionContext?.inputItems to OCR + draft creation.
        extensionContext?.completeRequest(returningItems: nil)
    }

    override func configurationItems() -> [Any]! { [] }
}
