import { Body, Controller, Get, Post } from "@nestjs/common";
import { AuthService } from "./auth.service";
import { LoginDto, SignupDto } from "./dto/auth.dto";
import { UseGuards, Req } from "@nestjs/common";
import { JwtGuard } from "./jwt/jwt.guard";

@Controller("/auth")
export class AuthController {
    constructor(private readonly authService: AuthService) {}

    @Get('profile') 
    @UseGuards(JwtGuard)
    getProfile(@Req() req: { user: any} ) {
        return req.user;
    }

    @Get()
    sayHello(): string {
        return this.authService.sayHello();
    }

    @Post("/signup")
    signUp(@Body() bodyMessage: SignupDto) {
        return this.authService.signUp(bodyMessage);
    }

    @Post("/login")
    login(@Body() bodyMessage: LoginDto) {
        return this.authService.login(bodyMessage);
    }

    @Post('refresh')
    async refresh(@Body() body: { refreshToken: string }) {
        return this.authService.refresh(body.refreshToken);
    }

    @Post('logout')
    @UseGuards(JwtGuard)
    async logout(@Req() req: { user:any }) {
        const userId = req.user.sub;
        return this.authService.logout(userId);

    }
}