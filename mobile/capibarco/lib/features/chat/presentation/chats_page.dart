import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/status_view.dart';
import '../../auth/presentation/auth_controller.dart';
import 'chats_controller.dart';

class ChatsPage extends ConsumerStatefulWidget {
  const ChatsPage({super.key});

  @override
  ConsumerState<ChatsPage> createState() => _ChatsPageState();
}

class _ChatsPageState extends ConsumerState<ChatsPage> {
  @override
  void initState() {
    super.initState();
    Future<void>.microtask(
      () => ref.read(chatsControllerProvider.notifier).load(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(chatsControllerProvider);
    final currentProfileId =
        ref.watch(authControllerProvider).session?.user.id ?? '';

    return Scaffold(
      body: PageShell(
        child: RefreshIndicator(
          onRefresh: () => ref.read(chatsControllerProvider.notifier).load(),
          child: ListView(
            children: <Widget>[
              SectionHeader(
                title: l10n.chats,
                subtitle: 'Active adoption conversations.',
              ),
              const SizedBox(height: 16),
              if (state.isLoading && state.conversations.isEmpty)
                StatusView.loading(message: l10n.loading)
              else if (state.errorMessage != null &&
                  state.conversations.isEmpty)
                StatusView.message(
                  message: state.errorMessage!,
                  icon: Icons.error_outline_rounded,
                  action: FilledButton(
                    onPressed: () =>
                        ref.read(chatsControllerProvider.notifier).load(),
                    child: Text(l10n.retry),
                  ),
                )
              else if (state.conversations.isEmpty)
                StatusView.message(
                  message: l10n.emptyChats,
                  icon: Icons.forum_outlined,
                )
              else
                ...state.conversations.map((conversation) {
                  final counterpartId = conversation.counterpartProfileId(
                    currentProfileId,
                  );
                  final counterpartName =
                      state.counterpartNames[counterpartId]?.trim() ?? '';
                  final title = counterpartName.isNotEmpty
                      ? counterpartName
                      : (counterpartId.isEmpty
                            ? 'Chat with user'
                            : 'Chat with $counterpartId');
                  final destination = Uri(
                    path: '/chat/${conversation.id}',
                    queryParameters: <String, String>{
                      'return_to': '/chats',
                      if (counterpartName.isNotEmpty) 'title': counterpartName,
                    },
                  ).toString();
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: SoftCard(
                      child: ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: const CircleAvatar(
                          child: Icon(Icons.chat_bubble_outline_rounded),
                        ),
                        title: Text(
                          title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        subtitle: Text(
                          'Ready to message',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        trailing: const Icon(Icons.chevron_right_rounded),
                        onTap: () => context.go(destination),
                      ),
                    ),
                  );
                }),
            ],
          ),
        ),
      ),
    );
  }
}
