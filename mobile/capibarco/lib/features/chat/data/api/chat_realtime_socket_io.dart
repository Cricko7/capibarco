import 'dart:io';

import 'chat_realtime_socket.dart';

Future<ChatRealtimeSocket> openChatRealtimeSocket(
  Uri uri, {
  required Map<String, dynamic> headers,
}) async {
  final socket = await WebSocket.connect(uri.toString(), headers: headers);
  socket.pingInterval = const Duration(seconds: 20);
  return _IoChatRealtimeSocket(socket);
}

class _IoChatRealtimeSocket implements ChatRealtimeSocket {
  const _IoChatRealtimeSocket(this._socket);

  final WebSocket _socket;

  @override
  Stream<dynamic> get messages => _socket;

  @override
  void sendText(String payload) {
    _socket.add(payload);
  }

  @override
  Future<void> close() async {
    await _socket.close();
  }
}
