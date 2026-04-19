import 'dart:async';

import 'package:capibarco/app/localization/app_localizations.dart';
import 'package:capibarco/app/theme/app_theme.dart';
import 'package:capibarco/core/config/environment.dart';
import 'package:capibarco/core/error/error_mapper.dart';
import 'package:capibarco/core/network/rest_service_client.dart';
import 'package:capibarco/features/chat/data/api/chat_api_client.dart';
import 'package:capibarco/features/chat/data/datasources/chat_remote_data_source.dart';
import 'package:capibarco/features/chat/data/repositories/chat_repository_impl.dart';
import 'package:capibarco/features/chat/domain/entities/chat_message.dart';
import 'package:capibarco/features/chat/presentation/chat_page.dart';
import 'package:capibarco/features/chat/presentation/chat_repository_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeChatRepository extends ChatRepositoryImpl {
  _FakeChatRepository()
    : super(
        remoteDataSource: ChatRemoteDataSource(_NoopChatApiClient()),
        errorMapper: const ErrorMapper(),
      );

  final StreamController<ChatMessageEntity> _realtimeController =
      StreamController<ChatMessageEntity>.broadcast();

  @override
  Future<List<ChatMessageEntity>> listMessages(String conversationId) async {
    return <ChatMessageEntity>[
      ChatMessageEntity(
        id: 'message-1',
        senderProfileId: 'profile-2',
        text: 'Привет',
        sentAt: DateTime.utc(2026, 4, 19, 9, 0),
      ),
    ];
  }

  @override
  Future<ChatMessageEntity> sendMessage({
    required String conversationId,
    required String text,
  }) async {
    return ChatMessageEntity(
      id: 'message-2',
      senderProfileId: 'profile-1',
      text: text,
      sentAt: DateTime.utc(2026, 4, 19, 9, 1),
    );
  }

  @override
  Stream<ChatMessageEntity> watchMessages({
    required String conversationId,
    required String accessToken,
  }) {
    return _realtimeController.stream;
  }

  void emitRealtimeMessage(ChatMessageEntity message) {
    _realtimeController.add(message);
  }

  Future<void> dispose() async {
    await _realtimeController.close();
  }
}

class _NoopChatApiClient extends ChatApiClient {
  _NoopChatApiClient()
    : super(
        RestServiceClient(
          dio: Dio(),
          config: const ServiceConfig(
            baseUrl: 'https://example.test',
            apiVersion: 'v1',
            protocol: TransportProtocol.rest,
          ),
        ),
      );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  tearDown(() async {
    final view =
        TestWidgetsFlutterBinding.instance.platformDispatcher.views.first;
    view.resetPhysicalSize();
    view.resetDevicePixelRatio();
  });

  Future<void> pumpChatPage(
    WidgetTester tester, {
    required Size size,
    _FakeChatRepository? repository,
  }) async {
    final view = tester.view;
    view.physicalSize = size;
    view.devicePixelRatio = 1;
    final chatRepository = repository ?? _FakeChatRepository();
    addTearDown(chatRepository.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [chatRepositoryProvider.overrideWithValue(chatRepository)],
        child: MaterialApp(
          theme: AppTheme.light(),
          localizationsDelegates: const <LocalizationsDelegate<dynamic>>[
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ChatPage(
            conversationId: 'conversation-1',
            title: 'Adoption chat',
            returnTo: '/chats',
          ),
        ),
      ),
    );

    await tester.pumpAndSettle();
  }

  testWidgets('keeps chat composer visible on desktop viewport', (
    tester,
  ) async {
    await pumpChatPage(tester, size: const Size(1280, 900));

    final composerField = find.byType(TextField);
    expect(composerField, findsOneWidget);

    final composerRect = tester.getRect(composerField);
    expect(composerRect.bottom, lessThanOrEqualTo(900));
    expect(composerRect.width, greaterThan(300));
  });

  testWidgets('stacks chat composer on narrow viewport', (tester) async {
    await pumpChatPage(tester, size: const Size(390, 844));

    final composerField = find.byType(TextField);
    final sendButton = find.widgetWithText(FilledButton, 'Send');

    expect(composerField, findsOneWidget);
    expect(sendButton, findsOneWidget);

    final composerRect = tester.getRect(composerField);
    final sendButtonRect = tester.getRect(sendButton);
    expect(composerRect.bottom, lessThan(sendButtonRect.top));
  });

  testWidgets('shows incoming realtime message without reload', (tester) async {
    final repository = _FakeChatRepository();

    await pumpChatPage(
      tester,
      size: const Size(390, 844),
      repository: repository,
    );

    repository.emitRealtimeMessage(
      ChatMessageEntity(
        id: 'message-3',
        senderProfileId: 'profile-2',
        text: 'Новое realtime сообщение',
        sentAt: DateTime.utc(2026, 4, 19, 9, 2),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Новое realtime сообщение'), findsOneWidget);
  });
}
