import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../auth/presentation/auth_controller.dart';
import '../data/api/chat_api_client.dart';
import '../data/datasources/chat_remote_data_source.dart';
import '../data/repositories/chat_repository_impl.dart';

final chatRepositoryProvider = Provider<ChatRepositoryImpl>((ref) {
  final environment = ref.watch(appEnvironmentProvider);
  return ChatRepositoryImpl(
    remoteDataSource: ChatRemoteDataSource(
      ChatApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.chat),
        ),
      ),
    ),
    errorMapper: ref.watch(errorMapperProvider),
  );
});
