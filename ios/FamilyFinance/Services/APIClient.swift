import Foundation

struct APIConfig {
    static var baseURL: URL {
        URL(string: UserDefaults.standard.string(forKey: "apiBaseURL") ?? "http://localhost:8080")!
    }
    static var apiKey: String {
        UserDefaults.standard.string(forKey: "apiKey") ?? "dev-api-key"
    }
}

struct MemberDTO: Codable, Identifiable {
    let id: String
    let name: String
}

struct AccountDTO: Codable, Identifiable {
    let id: String
    let name: String
    let type: String
}

struct CategoryDTO: Codable, Identifiable {
    let id: String
    let name: String
}

struct CreateTransactionRequest: Codable {
    let account_id: String
    let category_id: String?
    let paid_by_member_id: String
    let transaction_date: String
    let description: String
    let amount_satang: Int
}

final class APIClient {
    static let shared = APIClient()

    private func request<T: Decodable>(_ path: String) async throws -> T {
        var req = URLRequest(url: APIConfig.baseURL.appendingPathComponent(path))
        req.setValue(APIConfig.apiKey, forHTTPHeaderField: "X-API-Key")
        let (data, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.badServerResponse)
        }
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func post<T: Decodable, B: Encodable>(_ path: String, body: B) async throws -> T {
        var req = URLRequest(url: APIConfig.baseURL.appendingPathComponent(path))
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue(APIConfig.apiKey, forHTTPHeaderField: "X-API-Key")
        req.httpBody = try JSONEncoder().encode(body)
        let (data, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.badServerResponse)
        }
        return try JSONDecoder().decode(T.self, from: data)
    }

    func fetchMembers() async throws -> [MemberDTO] {
        try await request("/api/v1/members")
    }

    func fetchAccounts() async throws -> [AccountDTO] {
        try await request("/api/v1/accounts")
    }

    func fetchCategories() async throws -> [CategoryDTO] {
        try await request("/api/v1/categories")
    }

    func createTransaction(_ body: CreateTransactionRequest) async throws {
        let _: EmptyResponse = try await post("/api/v1/transactions", body: body)
    }
}

private struct EmptyResponse: Codable {}
