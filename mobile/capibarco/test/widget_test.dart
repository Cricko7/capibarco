import 'package:capibarco/main.dart';
import 'package:capibarco/src/domain/entities/session.dart';
import 'package:capibarco/src/domain/repositories/auth_repository.dart';
import 'package:capibarco/src/domain/usecases/check_backend_readiness_usecase.dart';
import 'package:capibarco/src/domain/usecases/login_usecase.dart';
import 'package:capibarco/src/presentation/login_controller.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthRepository implements AuthRepository {
  @override
  Future<bool> checkBackendReadiness() async => true;

  @override
  Future<Session> login({
    required String tenantId,
    required String email,
    required String password,
  }) async {
    return Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      email: email,
    );
  }
}

void main() {
  testWidgets('renders login page', (tester) async {
    final repository = _FakeAuthRepository();
    final controller = LoginController(
      loginUseCase: LoginUseCase(repository),
      checkBackendReadinessUseCase: CheckBackendReadinessUseCase(repository),
    );

    await tester.pumpWidget(CapibarcoApp(controller: controller));

    expect(find.text('Capibarco Auth'), findsOneWidget);
    expect(find.text('Login'), findsOneWidget);
  });
}
