import Foundation
import Vision
import UIKit

struct OCRResult {
    var amountSatang: Int?
    var date: Date?
    var merchant: String?
}

enum OCRService {
    static func parseReceipt(image: UIImage) async -> OCRResult {
        guard let cgImage = image.cgImage else { return OCRResult() }

        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.usesLanguageCorrection = true

        let handler = VNImageRequestHandler(cgImage: cgImage)
        try? handler.perform([request])

        guard let observations = request.results else { return OCRResult() }
        let fullText = observations.compactMap { $0.topCandidates(1).first?.string }.joined(separator: "\n")

        return OCRResult(
            amountSatang: extractAmount(from: fullText),
            date: extractDate(from: fullText),
            merchant: extractMerchant(from: fullText)
        )
    }

    private static func extractAmount(from text: String) -> Int? {
        let patterns = [
            #"฿\s*([\d,]+\.?\d*)"#,
            #"THB\s*([\d,]+\.?\d*)"#,
            #"([\d,]+\.?\d*)\s*บาท"#
        ]
        for pattern in patterns {
            if let match = text.range(of: pattern, options: .regularExpression) {
                let snippet = String(text[match])
                let digits = snippet.replacingOccurrences(of: #"[^\d.]"#, with: "", options: .regularExpression)
                if let value = Double(digits) {
                    return Int(value * 100)
                }
            }
        }
        return nil
    }

    private static func extractDate(from text: String) -> Date? {
        let detector = try? NSDataDetector(types: NSTextCheckingResult.CheckingType.date.rawValue)
        let range = NSRange(text.startIndex..., in: text)
        return detector?.firstMatch(in: text, range: range)?.date
    }

    private static func extractMerchant(from text: String) -> String? {
        let lines = text.components(separatedBy: .newlines).map { $0.trimmingCharacters(in: .whitespaces) }
        return lines.first { !$0.isEmpty && $0.count > 2 }
    }
}
