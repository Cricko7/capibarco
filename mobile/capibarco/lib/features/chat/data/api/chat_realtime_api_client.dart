import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;

import 'package:uuid/uuid.dart';

import '../dtos/chat_message_dto.dart';
import 'chat_realtime_socket.dart';
import 'chat_realtime_socket_stub.dart'
    if (dart.library.io) 'chat_realtime_socket_io.dart';

class ChatRealtimeApiClient {
  const ChatRealtimeApiClient({required String gatewayBaseUrl})
    : _gatewayBaseUrl = gatewayBaseUrl;

  final String _gatewayBaseUrl;

  static const _uuid = Uuid();

  Stream<ChatMessageDto> watchMessages({
    required String conversationId,
    required String accessToken,
  }) {
    if (accessToken.trim().isEmpty) {
      return const Stream<ChatMessageDto>.empty();
    }

    late final StreamController<ChatMessageDto> controller;
    ChatRealtimeSocket? socket;
    StreamSubscription<dynamic>? socketSubscription;
    Timer? reconnectTimer;
    var reconnectAttempts = 0;
    var isDisposed = false;
    late Future<void> Function() connect;

    Future<void> closeSocket() async {
      reconnectTimer?.cancel();
      reconnectTimer = null;
      await socketSubscription?.cancel();
      socketSubscription = null;
      final currentSocket = socket;
      socket = null;
      if (currentSocket != null) {
        await currentSocket.close();
      }
    }

    void scheduleReconnect() {
      if (isDisposed || reconnectTimer != null) {
        return;
      }

      unawaited(closeSocket());
      reconnectAttempts += 1;
      final delay = Duration(
        seconds: math.min(8, 1 << math.min(reconnectAttempts - 1, 3)),
      );
      reconnectTimer = Timer(delay, () {
        reconnectTimer = null;
        unawaited(connect());
      });
    }

    connect = () async {
      if (isDisposed) {
        return;
      }

      try {
        final openedSocket = await openChatRealtimeSocket(
          _buildSocketUri(),
          headers: <String, dynamic>{'Authorization': 'Bearer $accessToken'},
        );
        if (isDisposed) {
          await openedSocket.close();
          return;
        }

        socket = openedSocket;
        reconnectAttempts = 0;
        openedSocket.sendText(
          jsonEncode(
            _authFrame(
              accessToken: accessToken,
              conversationId: conversationId,
            ),
          ),
        );
        socketSubscription = openedSocket.messages.listen(
          (dynamic payload) {
            final message = _parseMessage(
              payload: payload,
              conversationId: conversationId,
            );
            if (message != null && !controller.isClosed) {
              controller.add(message);
            }
          },
          onDone: () {
            if (!isDisposed) {
              scheduleReconnect();
            }
          },
          onError: (Object error, StackTrace stackTrace) {
            if (!isDisposed) {
              scheduleReconnect();
            }
          },
          cancelOnError: false,
        );
      } catch (_) {
        if (isDisposed) {
          return;
        }
        scheduleReconnect();
      }
    };

    controller = StreamController<ChatMessageDto>(
      onListen: () {
        unawaited(connect());
      },
      onCancel: () async {
        isDisposed = true;
        await closeSocket();
      },
    );

    return controller.stream;
  }

  Uri _buildSocketUri() {
    final baseUri = Uri.parse(_gatewayBaseUrl);
    final normalizedPath = baseUri.path.isEmpty || baseUri.path == '/'
        ? '/ws/chat'
        : '${baseUri.path.replaceFirst(RegExp(r'/$'), '')}/ws/chat';
    final scheme = switch (baseUri.scheme) {
      'https' => 'wss',
      'http' => 'ws',
      _ => baseUri.scheme,
    };

    return baseUri.replace(
      scheme: scheme,
      path: normalizedPath,
      queryParameters: null,
      fragment: null,
    );
  }

  Map<String, dynamic> _authFrame({
    required String accessToken,
    required String conversationId,
  }) {
    return <String, dynamic>{
      'frame_id': _uuid.v4(),
      'type': 'CHAT_FRAME_TYPE_AUTH',
      'client_sent_at': DateTime.now().toUtc().toIso8601String(),
      'auth': <String, dynamic>{
        'access_token': accessToken,
        'conversation_id': conversationId,
      },
    };
  }

  ChatMessageDto? _parseMessage({
    required dynamic payload,
    required String conversationId,
  }) {
    final decodedPayload = switch (payload) {
      String value => value,
      List<int> value => utf8.decode(value),
      _ => payload.toString(),
    };
    final frame = jsonDecode(decodedPayload) as Map<String, dynamic>;
    final event = frame['event'] as Map<String, dynamic>?;
    final messageSent = event?['message_sent'] as Map<String, dynamic>?;
    final message = messageSent?['message'] as Map<String, dynamic>?;
    if (message == null) {
      return null;
    }
    if ((message['conversation_id'] as String? ?? '') != conversationId) {
      return null;
    }
    return ChatMessageDto.fromJson(message);
  }
}
