import 'package:capibarco/features/chat/data/api/chat_realtime_api_client.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('builds websocket uri with access token query parameter', () {
    const client = ChatRealtimeApiClient(
      gatewayBaseUrl: 'http://localhost:18088',
    );

    final uri = client.buildSocketUriForTesting('token-123');

    expect(uri.scheme, 'ws');
    expect(uri.path, '/ws/chat');
    expect(uri.queryParameters['access_token'], 'token-123');
  });
}
