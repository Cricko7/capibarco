import 'chat_realtime_socket.dart';

Future<ChatRealtimeSocket> openChatRealtimeSocket(
  Uri uri, {
  required Map<String, dynamic> headers,
}) async {
  throw UnsupportedError('Chat realtime is not supported on this platform.');
}
