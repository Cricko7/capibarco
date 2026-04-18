import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/localization/app_localizations.dart';
import '../../billing/presentation/donate_animal_sheet.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/stale_banner.dart';
import '../../../shared/presentation/status_view.dart';
import 'feed_controller.dart';

class FeedPage extends ConsumerStatefulWidget {
  const FeedPage({super.key});

  @override
  ConsumerState<FeedPage> createState() => _FeedPageState();
}

class _FeedPageState extends ConsumerState<FeedPage> {
  @override
  void initState() {
    super.initState();
    Future<void>.microtask(
      () => ref.read(feedControllerProvider.notifier).load(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(feedControllerProvider);

    if (state.isLoading && state.cards.isEmpty) {
      return Scaffold(body: StatusView.loading(message: l10n.loading));
    }

    if (state.errorMessage != null && state.cards.isEmpty) {
      return Scaffold(
        body: StatusView.message(
          message: state.errorMessage!,
          icon: Icons.error_outline_rounded,
          action: FilledButton(
            onPressed: () => ref.read(feedControllerProvider.notifier).load(),
            child: Text(l10n.retry),
          ),
        ),
      );
    }

    return Scaffold(
      body: PageShell(
        child: RefreshIndicator(
          onRefresh: () => ref.read(feedControllerProvider.notifier).load(),
          child: ListView(
            children: <Widget>[
              SectionHeader(
                title: l10n.feed,
                subtitle: 'Swipe-ready feed powered by the PetMatch gateway.',
              ),
              if (state.isStale) ...<Widget>[
                const SizedBox(height: 16),
                StaleBanner(message: l10n.staleData),
              ],
              const SizedBox(height: 16),
              if (state.cards.isEmpty)
                StatusView.message(
                  message: l10n.emptyFeed,
                  icon: Icons.pets_outlined,
                )
              else
                ...state.cards.map(
                  (card) => Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: SoftCard(
                      padding: EdgeInsets.zero,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          ClipRRect(
                            borderRadius: const BorderRadius.vertical(
                              top: Radius.circular(28),
                            ),
                            child: SizedBox(
                              height: 220,
                              width: double.infinity,
                              child: card.photoUrl.isNotEmpty
                                  ? Image.network(
                                      card.photoUrl,
                                      fit: BoxFit.cover,
                                    )
                                  : DecoratedBox(
                                      decoration: BoxDecoration(
                                        gradient: LinearGradient(
                                          colors: <Color>[
                                            Theme.of(
                                              context,
                                            ).colorScheme.primaryContainer,
                                            Theme.of(
                                              context,
                                            ).colorScheme.secondaryContainer,
                                          ],
                                        ),
                                      ),
                                      child: Icon(
                                        Icons.pets_rounded,
                                        size: 52,
                                        color: Theme.of(
                                          context,
                                        ).colorScheme.primary,
                                      ),
                                    ),
                            ),
                          ),
                          Padding(
                            padding: const EdgeInsets.all(20),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: <Widget>[
                                Row(
                                  children: <Widget>[
                                    Expanded(
                                      child: Text(
                                        '${card.name}, ${card.speciesLabel}',
                                        style: Theme.of(context)
                                            .textTheme
                                            .headlineSmall
                                            ?.copyWith(
                                              fontWeight: FontWeight.w900,
                                            ),
                                      ),
                                    ),
                                    if (card.boosted)
                                      Chip(
                                        avatar: const Icon(
                                          Icons.bolt_rounded,
                                          size: 18,
                                        ),
                                        label: const Text('Boosted'),
                                      ),
                                  ],
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  '${card.ownerDisplayName}${card.city.isNotEmpty ? ' · ${card.city}' : ''}',
                                  style: Theme.of(
                                    context,
                                  ).textTheme.titleMedium,
                                ),
                                const SizedBox(height: 12),
                                Text(
                                  card.description.isEmpty
                                      ? 'No description yet.'
                                      : card.description,
                                ),
                                if (card.rankingReasons.isNotEmpty) ...<Widget>[
                                  const SizedBox(height: 14),
                                  Wrap(
                                    spacing: 8,
                                    runSpacing: 8,
                                    children: card.rankingReasons
                                        .take(3)
                                        .map(
                                          (reason) => Chip(label: Text(reason)),
                                        )
                                        .toList(),
                                  ),
                                ],
                                const SizedBox(height: 14),
                                Align(
                                  alignment: Alignment.centerLeft,
                                  child: TextButton.icon(
                                    onPressed: () => showModalBottomSheet<void>(
                                      context: context,
                                      isScrollControlled: true,
                                      backgroundColor: Theme.of(
                                        context,
                                      ).colorScheme.surfaceContainerLowest,
                                      shape: const RoundedRectangleBorder(
                                        borderRadius: BorderRadius.vertical(
                                          top: Radius.circular(32),
                                        ),
                                      ),
                                      builder: (context) => DonateAnimalSheet(
                                        animalId: card.animalId,
                                        animalName: card.name,
                                        ownerDisplayName: card.ownerDisplayName,
                                      ),
                                    ),
                                    icon: const Icon(
                                      Icons.volunteer_activism_rounded,
                                    ),
                                    label: Text(l10n.supportAnimal),
                                  ),
                                ),
                                const SizedBox(height: 18),
                                Row(
                                  children: <Widget>[
                                    Expanded(
                                      child: OutlinedButton.icon(
                                        onPressed: () => ref
                                            .read(
                                              feedControllerProvider.notifier,
                                            )
                                            .swipe(card: card, liked: false),
                                        icon: const Icon(Icons.close_rounded),
                                        label: Text(l10n.pass),
                                      ),
                                    ),
                                    const SizedBox(width: 12),
                                    Expanded(
                                      child: FilledButton.icon(
                                        onPressed: () => ref
                                            .read(
                                              feedControllerProvider.notifier,
                                            )
                                            .swipe(card: card, liked: true),
                                        icon: const Icon(
                                          Icons.favorite_rounded,
                                        ),
                                        label: Text(l10n.like),
                                      ),
                                    ),
                                  ],
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              if (state.nextPageToken.isNotEmpty) ...<Widget>[
                const SizedBox(height: 8),
                FilledButton.tonal(
                  onPressed: state.isLoadingMore
                      ? null
                      : () => ref
                            .read(feedControllerProvider.notifier)
                            .loadMore(),
                  child: state.isLoadingMore
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2.4),
                        )
                      : const Text('Load more'),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
