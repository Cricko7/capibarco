import 'package:capibarco/app/localization/app_localizations.dart';
import 'package:capibarco/app/theme/app_theme.dart';
import 'package:capibarco/features/auth/domain/entities/auth_session.dart';
import 'package:capibarco/features/auth/presentation/auth_controller.dart';
import 'package:capibarco/features/auth/presentation/auth_state.dart';
import 'package:capibarco/features/chat/domain/entities/chat_conversation.dart';
import 'package:capibarco/features/chat/presentation/chats_controller.dart';
import 'package:capibarco/features/chat/presentation/chats_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthController extends AuthController {
  @override
  AuthState build() {
    return AuthState(
      status: AuthStatus.authenticated,
      session: AuthSession(
        user: const AuthUser(
          id: 'profile-1',
          tenantId: 'default',
          email: 'alice@example.com',
          isActive: true,
        ),
        accessToken: 'token',
        refreshToken: 'refresh',
        expiresAt: DateTime.utc(2026, 4, 30),
      ),
      isBootstrapping: false,
    );
  }
}

class _FakeChatsController extends ChatsController {
  @override
  ChatsState build() {
    return const ChatsState(
      conversations: <ChatConversationEntity>[
        ChatConversationEntity(
          id: 'conversation-1',
          adopterProfileId: 'profile-1',
          ownerProfileId: 'profile-2',
        ),
      ],
      counterpartNames: <String, String>{'profile-2': 'Mila Shelter'},
    );
  }

  @override
  Future<void> load() async {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('shows counterpart display name instead of raw profile id', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_FakeAuthController.new),
          chatsControllerProvider.overrideWith(_FakeChatsController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.light(),
          localizationsDelegates: const <LocalizationsDelegate<dynamic>>[
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ChatsPage(),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Mila Shelter'), findsOneWidget);
    expect(find.text('Chat with profile-2'), findsNothing);
  });
}
