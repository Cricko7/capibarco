// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:async';
import 'dart:html' as html;

import 'chat_realtime_socket.dart';

Future<ChatRealtimeSocket> openChatRealtimeSocket(
  Uri uri, {
  required Map<String, dynamic> headers,
}) async {
  final socket = html.WebSocket(uri.toString());
  final connected = Completer<void>();

  late final StreamSubscription<html.Event> openSubscription;
  late final StreamSubscription<html.Event> errorSubscription;

  openSubscription = socket.onOpen.listen((_) {
    if (!connected.isCompleted) {
      connected.complete();
    }
  });
  errorSubscription = socket.onError.listen((_) {
    if (!connected.isCompleted) {
      connected.completeError(
        StateError('Failed to connect to chat realtime socket.'),
      );
    }
  });

  try {
    await connected.future.timeout(const Duration(seconds: 10));
    return _WebChatRealtimeSocket(socket);
  } finally {
    await openSubscription.cancel();
    await errorSubscription.cancel();
  }
}

class _WebChatRealtimeSocket implements ChatRealtimeSocket {
  _WebChatRealtimeSocket(this._socket) {
    _messageSubscription = _socket.onMessage.listen((event) {
      if (!_controller.isClosed) {
        _controller.add(event.data);
      }
    });
    _errorSubscription = _socket.onError.listen((_) {
      if (!_controller.isClosed) {
        _controller.addError(StateError('Chat realtime socket error.'));
      }
    });
    _closeSubscription = _socket.onClose.listen((_) {
      if (!_controller.isClosed) {
        unawaited(_controller.close());
      }
    });
  }

  final html.WebSocket _socket;
  final StreamController<dynamic> _controller =
      StreamController<dynamic>.broadcast();

  late final StreamSubscription<html.MessageEvent> _messageSubscription;
  late final StreamSubscription<html.Event> _errorSubscription;
  late final StreamSubscription<html.Event> _closeSubscription;

  @override
  Stream<dynamic> get messages => _controller.stream;

  @override
  void sendText(String payload) {
    _socket.send(payload);
  }

  @override
  Future<void> close() async {
    await _messageSubscription.cancel();
    await _errorSubscription.cancel();
    await _closeSubscription.cancel();
    if (_socket.readyState == html.WebSocket.CONNECTING ||
        _socket.readyState == html.WebSocket.OPEN) {
      _socket.close();
    }
    if (!_controller.isClosed) {
      await _controller.close();
    }
  }
}
